package track

import (
	"math"
	"testing"
)

// metersToPoints строит контур из смещений в метрах вокруг заданной широты —
// так тестовые фигуры задаются в понятных единицах, а не в градусах.
func metersToPoints(lat0, lng0 float64, offsets [][2]float64) []Point {
	mLng := metersPerDegreeLat * math.Cos(lat0*math.Pi/180)
	out := make([]Point, len(offsets))
	for i, o := range offsets {
		out[i] = Point{
			Lat: lat0 + o[1]/metersPerDegreeLat,
			Lng: lng0 + o[0]/mLng,
		}
	}
	return out
}

func TestComputeSquare(t *testing.T) {
	// Квадрат 200×200 м, обход против часовой: периметр 800 м, 4 поворота по 90°,
	// самая длинная прямая — одна сторона.
	square := metersToPoints(55.75, 37.61, [][2]float64{{0, 0}, {200, 0}, {200, 200}, {0, 200}})

	got := Compute(square)

	if math.Abs(float64(got.LengthM)-800) > 2 {
		t.Errorf("LengthM = %d, ожидали ~800", got.LengthM)
	}
	if got.Corners != 4 {
		t.Errorf("Corners = %d, ожидали 4", got.Corners)
	}
	if got.Direction != CounterClockwise {
		t.Errorf("Direction = %q, ожидали %q", got.Direction, CounterClockwise)
	}
	if math.Abs(float64(got.MainStraightM)-200) > 2 {
		t.Errorf("MainStraightM = %d, ожидали ~200", got.MainStraightM)
	}
}

func TestComputeDirectionIsReversible(t *testing.T) {
	square := metersToPoints(55.75, 37.61, [][2]float64{{0, 0}, {200, 0}, {200, 200}, {0, 200}})
	reversed := make([]Point, len(square))
	for i, p := range square {
		reversed[len(square)-1-i] = p
	}

	if dir := Compute(square).Direction; dir != CounterClockwise {
		t.Errorf("исходный обход: Direction = %q, ожидали %q", dir, CounterClockwise)
	}
	if dir := Compute(reversed).Direction; dir != Clockwise {
		t.Errorf("обратный обход: Direction = %q, ожидали %q", dir, Clockwise)
	}
}

func TestComputeIgnoresClosingDuplicate(t *testing.T) {
	// Геометрия трасс приходит замкнутой явно: последняя точка дублирует первую.
	// Дубль не должен добавлять нулевой сегмент и ложный поворот.
	open := metersToPoints(55.75, 37.61, [][2]float64{{0, 0}, {200, 0}, {200, 200}, {0, 200}})
	closed := append(append([]Point{}, open...), open[0])

	if Compute(open) != Compute(closed) {
		t.Errorf("замкнутый дублем контур дал другой результат:\nбез дубля %+v\nс дублем %+v",
			Compute(open), Compute(closed))
	}
}

func TestComputeStraightIgnoresTinyJitter(t *testing.T) {
	// Длинная сторона, разбитая на промежуточные точки с микроскопическим отклонением,
	// обязана остаться одной прямой, а не рассыпаться на короткие куски.
	jittered := metersToPoints(55.75, 37.61, [][2]float64{
		{0, 0}, {100, 0.3}, {200, 0}, {300, 0.3}, {400, 0},
		{400, 200}, {0, 200},
	})

	got := Compute(jittered)

	if got.MainStraightM < 390 {
		t.Errorf("MainStraightM = %d, ожидали ~400: микроотклонения не должны резать прямую", got.MainStraightM)
	}
	if got.Corners != 4 {
		t.Errorf("Corners = %d, ожидали 4: микроотклонения не должны считаться поворотами", got.Corners)
	}
}

func TestComputeCornerOnArrayBoundary(t *testing.T) {
	// Поворот, лежащий на стыке конца и начала массива, должен считаться один раз.
	// Контур тот же квадрат, но нумерация начата прямо в вершине.
	fromCorner := metersToPoints(55.75, 37.61, [][2]float64{{200, 0}, {200, 200}, {0, 200}, {0, 0}})

	if got := Compute(fromCorner).Corners; got != 4 {
		t.Errorf("Corners = %d, ожидали 4", got)
	}
}

func TestComputeTooFewPoints(t *testing.T) {
	for _, tc := range []struct {
		name   string
		points []Point
	}{
		{"пусто", nil},
		{"одна точка", []Point{{Lat: 55.75, Lng: 37.61}}},
		{"две точки", []Point{{Lat: 55.75, Lng: 37.61}, {Lat: 55.76, Lng: 37.62}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Compute(tc.points); got != (Stats{}) {
				t.Errorf("Compute(%v) = %+v, ожидали нули", tc.points, got)
			}
		})
	}
}

func TestComputeRealTracks(t *testing.T) {
	// Контрольные значения — те же, что заложены в миграцию 00006 и в док F5.
	// Тест страхует от расхождения витрины с реальным контуром.
	tests := []struct {
		name          string
		points        []Point
		lengthM       int
		corners       int
		direction     Direction
		mainStraightM int
	}{
		{
			name:          "Городское кольцо",
			points:        gorodskoeKoltso,
			lengthM:       980,
			corners:       10,
			direction:     Clockwise,
			mainStraightM: 200,
		},
		{
			name:          "Спортивная трасса",
			points:        sportivnayaTrassa,
			lengthM:       1240,
			corners:       16,
			direction:     CounterClockwise,
			mainStraightM: 237,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.points)

			if diff := got.LengthM - tc.lengthM; diff < -10 || diff > 10 {
				t.Errorf("LengthM = %d, ожидали ~%d", got.LengthM, tc.lengthM)
			}
			if got.Corners != tc.corners {
				t.Errorf("Corners = %d, ожидали %d", got.Corners, tc.corners)
			}
			if got.Direction != tc.direction {
				t.Errorf("Direction = %q, ожидали %q", got.Direction, tc.direction)
			}
			if diff := got.MainStraightM - tc.mainStraightM; diff < -10 || diff > 10 {
				t.Errorf("MainStraightM = %d, ожидали ~%d", got.MainStraightM, tc.mainStraightM)
			}
		})
	}
}
