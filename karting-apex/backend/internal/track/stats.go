// Package track считает характеристики трассы из её геометрии (F5).
//
// Длина круга, число поворотов, направление вращения и главная прямая намеренно
// НЕ хранятся в БД: они выводятся из того же контура, который рисуется на схеме,
// поэтому не могут с ним разъехаться. В routes хранится только «паспорт» —
// покрытие, ширина, перепад высот и прочее, что из геометрии не выводится.
package track

import "math"

// Point — точка контура трассы в градусах.
type Point struct {
	Lat float64
	Lng float64
}

// Direction — направление прохождения круга.
type Direction string

const (
	Clockwise        Direction = "clockwise"
	CounterClockwise Direction = "counter_clockwise"
)

// Stats — характеристики, выведенные из контура.
type Stats struct {
	LengthM       int       // длина круга, м
	Corners       int       // количество поворотов
	Direction     Direction // направление вращения
	MainStraightM int       // длина самого длинного прямого участка, м
}

const (
	// Поворотом считается связка подряд идущих вершин, повёрнутых в одну сторону
	// суммарно не меньше чем на столько градусов. Порог отсекает мелкие подруливания
	// на «прямых», которые появляются из-за дискретизации контура.
	cornerMinTotalDeg = 35.0
	// Вершины с поворотом меньше этого считаются шумом и не начинают связку.
	cornerDeadZoneDeg = 3.0
	// Связка закрывается, когда после последнего поворота накопилось столько метров
	// прямой. Именно прямая разделяет два поворота, а не смена направления: у квадрата
	// все четыре поворота левые, но это четыре разных поворота. Считается накопленная
	// дистанция, а не длина одного сегмента: на равномерно разбитом контуре все сегменты
	// одинаковые, и по одному сегменту прямую от поворота не отличить.
	cornerSeparationM = 25.0
	// Участок считается прямым, пока поворот в вершине не превышает это значение.
	straightMaxTurnDeg = 6.0

	metersPerDegreeLat = 111320.0
)

// Compute возвращает характеристики замкнутого контура. Для контуров короче трёх
// точек возвращает нули: посчитать по ним нечего.
//
// Контур считается замкнутым (последняя точка соединена с первой). Если геометрия
// пришла с явно продублированной первой точкой в конце, дубль отбрасывается —
// иначе он даст лишний нулевой сегмент и ложный «поворот».
func Compute(points []Point) Stats {
	pts := project(dropClosingDuplicate(points))
	if len(pts) < 3 {
		return Stats{}
	}

	dir := CounterClockwise
	if signedArea(pts) < 0 {
		dir = Clockwise
	}

	return Stats{
		LengthM:       int(math.Round(perimeter(pts))),
		Corners:       countCorners(pts),
		Direction:     dir,
		MainStraightM: int(math.Round(longestStraight(pts))),
	}
}

// point2D — точка в локальных метрах: x на восток, y на север.
type point2D struct{ x, y float64 }

func dropClosingDuplicate(points []Point) []Point {
	if len(points) >= 2 {
		first, last := points[0], points[len(points)-1]
		if first.Lat == last.Lat && first.Lng == last.Lng {
			return points[:len(points)-1]
		}
	}
	return points
}

// project переводит градусы в метры вокруг средней широты контура. Долгота
// корректируется на cos(широты), иначе форма растянется по горизонтали и все
// расчёты (длина, углы) поедут.
func project(points []Point) []point2D {
	if len(points) == 0 {
		return nil
	}
	var sumLat float64
	for _, p := range points {
		sumLat += p.Lat
	}
	midLat := sumLat / float64(len(points))
	metersPerDegreeLng := metersPerDegreeLat * math.Cos(midLat*math.Pi/180)

	out := make([]point2D, len(points))
	for i, p := range points {
		out[i] = point2D{
			x: p.Lng * metersPerDegreeLng,
			y: p.Lat * metersPerDegreeLat,
		}
	}
	return out
}

func perimeter(pts []point2D) float64 {
	var total float64
	for i := range pts {
		total += segmentLen(pts, i)
	}
	return total
}

func segmentLen(pts []point2D, i int) float64 {
	a, b := pts[i], pts[(i+1)%len(pts)]
	return math.Hypot(b.x-a.x, b.y-a.y)
}

// turnAt — угол поворота в вершине i, градусы: положительный влево, отрицательный вправо.
func turnAt(pts []point2D, i int) float64 {
	n := len(pts)
	p0, p1, p2 := pts[(i-1+n)%n], pts[i], pts[(i+1)%n]
	d := (math.Atan2(p2.y-p1.y, p2.x-p1.x) - math.Atan2(p1.y-p0.y, p1.x-p0.x)) * 180 / math.Pi
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

// countCorners накапливает поворот по подряд идущим вершинам и закрывает связку,
// когда после неё пошла прямая длиной cornerSeparationM либо трасса повернула в
// другую сторону (шикана). Набравшая cornerMinTotalDeg связка считается поворотом.
//
// Обход начинается с вершины сразу после прямой — тогда ни одна связка не окажется
// разрезанной границей массива.
func countCorners(pts []point2D) int {
	n := len(pts)

	start := 0
	run := 0.0
	for i := 0; i < 2*n; i++ {
		idx := i % n
		if math.Abs(turnAt(pts, idx)) > cornerDeadZoneDeg {
			run = 0
		} else {
			run += segmentLen(pts, idx)
			if run >= cornerSeparationM {
				start = (idx + 1) % n
				break
			}
		}
	}

	count := 0
	acc, sign := 0.0, 0
	closeCorner := func() {
		if math.Abs(acc) >= cornerMinTotalDeg {
			count++
		}
		acc, sign = 0, 0
	}

	straightRun := 0.0
	for k := 0; k < n; k++ {
		i := (start + k) % n
		t := turnAt(pts, i)

		if math.Abs(t) > cornerDeadZoneDeg {
			s := 1
			if t < 0 {
				s = -1
			}
			if sign != 0 && s != sign {
				closeCorner() // сменилось направление — это уже следующий поворот
			}
			acc += t
			sign = s
			straightRun = 0
		}

		straightRun += segmentLen(pts, i)
		if straightRun >= cornerSeparationM {
			closeCorner()
		}
	}
	closeCorner()
	return count
}

// longestStraight — самая длинная цепочка сегментов, на которой трасса почти не
// поворачивает. Стартуем с каждой вершины: для замкнутого контура прямая может
// начинаться где угодно, в том числе на стыке конца и начала массива.
//
// Первый сегмент берётся всегда: отрезок между двумя вершинами прямой по определению,
// и поворот на входе в него — это предыдущий поворот, а не изгиб самой прямой.
// Ограничение проверяется только для вершин ВНУТРИ цепочки.
func longestStraight(pts []point2D) float64 {
	n := len(pts)
	var best float64
	for start := 0; start < n; start++ {
		acc := segmentLen(pts, start)
		for k := 1; k < n; k++ {
			i := (start + k) % n
			if math.Abs(turnAt(pts, i)) > straightMaxTurnDeg {
				break
			}
			acc += segmentLen(pts, i)
		}
		if acc > best {
			best = acc
		}
	}
	return best
}

// signedArea — знаковая площадь контура (формула шнурования). Знак даёт направление
// обхода: в системе «x на восток, y на север» отрицательная площадь — по часовой.
func signedArea(pts []point2D) float64 {
	var s float64
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		s += a.x*b.y - b.x*a.y
	}
	return s / 2
}
