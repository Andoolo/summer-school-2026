package auth

// OTPStats — сколько кодов выдано на номер и сколько раз коды вводили неверно,
// за последний час и за сутки. Считается по таблице otp_codes, поэтому переживает
// перезапуск сервиса и не обходится сменой IP.
//
// Неверные попытки относятся к окну по времени выдачи кода, а не по времени самой
// попытки: для оценки подбора этого достаточно, а отдельная таблица попыток не нужна.
type OTPStats struct {
	CodesLastHour  int
	CodesLastDay   int
	FailedLastHour int
	FailedLastDay  int
}

// OTPLimits — лимиты на один номер телефона.
//
// Без них подбор кода упирался только в «5 попыток на код» и «новый код раз в
// минуту»: 5 попыток в минуту без ограничения по времени, и 4-значный код к чужому
// номеру подбирался примерно за полдня. С лимитом 20 неверных попыток в сутки
// вероятность угадать за сутки — 0.2%.
type OTPLimits struct {
	CodesPerHour  int
	CodesPerDay   int
	FailedPerHour int
	FailedPerDay  int
}

// DefaultOTPLimits подобраны так, чтобы живой человек в них не упирался: при паузе
// 60 секунд между кодами пять кодов в час — это пять неудачных доставок подряд.
func DefaultOTPLimits() OTPLimits {
	return OTPLimits{
		CodesPerHour:  5,
		CodesPerDay:   10,
		FailedPerHour: 10,
		FailedPerDay:  20,
	}
}

// AllowRequest — можно ли выдать ещё один код.
func (l OTPLimits) AllowRequest(stats OTPStats) bool {
	return stats.CodesLastHour < l.CodesPerHour && stats.CodesLastDay < l.CodesPerDay
}

// AllowVerify — можно ли проверять код. После исчерпания лимита неверных попыток
// проверка не выполняется даже для правильного кода: иначе лимит не мешал бы подбору,
// а лишь замедлял бы его.
func (l OTPLimits) AllowVerify(stats OTPStats) bool {
	return stats.FailedLastHour < l.FailedPerHour && stats.FailedLastDay < l.FailedPerDay
}

// MaskPhone оставляет последние 4 цифры — для логов о срабатывании лимитов.
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "***" + phone[len(phone)-4:]
}
