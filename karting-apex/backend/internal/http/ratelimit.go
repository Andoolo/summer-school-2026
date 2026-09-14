package httpapi

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitOptions — лимиты запросов с одного IP. Это второй рубеж: главные лимиты на
// коды подтверждения считаются по номеру телефона в сервисе авторизации и сменой IP
// не обходятся. Здесь — защита от засыпания API и перебора номеров с одного адреса.
//
// Счётчики живут в памяти процесса: у сервиса один инстанс, а сброс при перезапуске
// лишь ненадолго ослабляет второй рубеж.
type RateLimitOptions struct {
	// TrustProxy — брать IP клиента из CF-Connecting-IP. Нужно за прокси (Render стоит
	// за Cloudflare): иначе все запросы приходят с адреса внутреннего балансировщика и
	// делят один лимит на всех. Без Cloudflare включать нельзя — заголовок подделывается.
	TrustProxy bool
	Logger     *slog.Logger
}

type rateRule struct {
	name    string
	limiter *windowLimiter
}

// Лимиты подобраны с запасом для живых пользователей: за одним мобильным IP (CGNAT)
// бывают сотни абонентов, поэтому запрос кода — 10 за 10 минут на адрес, а не 3.
func newRateRules() (general rateRule, byRoute map[string]rateRule) {
	requestCode := rateRule{name: "request_code", limiter: newWindowLimiter(10, 10*time.Minute)}
	verifyCode := rateRule{name: "verify_code", limiter: newWindowLimiter(30, 10*time.Minute)}
	// Каждый гостевой вход — новый аккаунт в базе; живому человеку хватит пары входов.
	demoLogin := rateRule{name: "demo_login", limiter: newWindowLimiter(5, time.Hour)}
	return rateRule{name: "general", limiter: newWindowLimiter(300, time.Minute)},
		map[string]rateRule{
			"POST /auth/request-code":          requestCode,
			"POST /profile/phone/request-code": requestCode,
			"POST /auth/verify-code":           verifyCode,
			"POST /profile/phone/confirm":      verifyCode,
			"POST /auth/demo":                  demoLogin,
		}
}

func rateLimitMiddleware(opts RateLimitOptions) func(http.Handler) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	general, byRoute := newRateRules()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Preflight и проверки живости не считаем: первые шлёт браузер сам,
			// вторые — платформа.
			if r.Method == http.MethodOptions || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r, opts.TrustProxy)

			if rule, ok := byRoute[r.Method+" "+r.URL.Path]; ok {
				if allowed, retryAfter := rule.limiter.allow(ip); !allowed {
					writeRateLimited(w, logger, rule.name, ip, r.URL.Path, retryAfter)
					return
				}
			}
			if allowed, retryAfter := general.limiter.allow(ip); !allowed {
				writeRateLimited(w, logger, general.name, ip, r.URL.Path, retryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeRateLimited(w http.ResponseWriter, logger *slog.Logger, rule, ip, path string, retryAfter time.Duration) {
	logger.Warn("rate limit exceeded", "rule", rule, "client_ip", ip, "path", path)
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
	WriteError(w, http.StatusTooManyRequests, CodeTooManyRequests, "Слишком много запросов. Повторите попытку позже.", nil)
}

// clientIP возвращает ключ клиента для лимитов. IPv6 сводится к префиксу /64: провайдер
// обычно выдаёт абоненту целую подсеть, и без этого каждый адрес из неё получал бы
// свой лимит.
//
// За прокси IP берётся из CF-Connecting-IP, а не из X-Forwarded-For. Проверено на
// production: Render дописывает свою цепочку к присланному клиентом X-Forwarded-For,
// поэтому первый адрес в нём подделывается и обходил лимит. CF-Connecting-IP
// выставляет Cloudflare перед Render: присланный клиентом такой заголовок Cloudflare
// отклоняет (403), а в обход Cloudflare сервис на Render недоступен.
//
// Если заголовка нет, берётся адрес соединения, а не X-Forwarded-For: общий лимит на
// всех хуже для пользователей, но не даёт обойти защиту подделкой.
func clientIP(r *http.Request, trustProxy bool) string {
	raw := ""
	if trustProxy {
		if connecting := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); net.ParseIP(connecting) != nil {
			raw = connecting
		}
	}
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}
	if ip.To4() == nil {
		return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
	}
	return ip.String()
}

// windowLimiter — счётчик запросов в фиксированном окне на ключ.
type windowLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	maxKeys   int
	entries   map[string]*windowEntry
	lastSweep time.Time
	now       func() time.Time
}

type windowEntry struct {
	count   int
	resetAt time.Time
}

func newWindowLimiter(limit int, window time.Duration) *windowLimiter {
	return &windowLimiter{
		limit:   limit,
		window:  window,
		maxKeys: 50_000,
		entries: make(map[string]*windowEntry),
		now:     time.Now,
	}
}

// allow засчитывает запрос и сообщает, укладывается ли он в лимит, а если нет —
// через сколько окно сбросится.
func (l *windowLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) >= l.window {
		l.sweep(now)
	}

	entry, ok := l.entries[key]
	if !ok || !now.Before(entry.resetAt) {
		if !ok && len(l.entries) >= l.maxKeys {
			// Таблица забита уникальными ключами (например, поток с подделанными
			// адресами). Сбрасываем её, а не отказываем всем подряд: блокировка всех
			// пользователей хуже временного ослабления второго рубежа.
			l.entries = make(map[string]*windowEntry)
		}
		entry = &windowEntry{resetAt: now.Add(l.window)}
		l.entries[key] = entry
	}

	entry.count++
	if entry.count > l.limit {
		return false, entry.resetAt.Sub(now)
	}
	return true, 0
}

func (l *windowLimiter) sweep(now time.Time) {
	for key, entry := range l.entries {
		if !now.Before(entry.resetAt) {
			delete(l.entries, key)
		}
	}
	l.lastSweep = now
}

// bodyLimitMiddleware ограничивает размер тела запроса: самый большой легальный запрос
// API — список времён кругов, это килобайты. Без ограничения огромный JSON целиком
// читался бы в память.
func bodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
