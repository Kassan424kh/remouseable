package lazypoint

import (
	"math"

	Point "github.com/kevinconway/remouseable/pkg/internal/robotgo/oob/point"
)

type LazyPoint struct {
	X, Y float64
}

/**
 * Update the x and y values
 *
 * @param {Point} point
 */
func (l LazyPoint) Update(p Point.Point) {
	l.X = p.X
	l.Y = p.Y
}

/**
 * Move the point to another position using an angle and distance
 *
 * @param {number} angle The angle in radians
 * @param {number} distance How much the point should be moved
 */
func (l LazyPoint) MoveByAngle(angle, distance float64) {
	// Rotate the angle based on the browser coordinate system ([0,0] in the top left)
	angleRotated := angle + (math.Pi / 2)

	l.X = l.X + (math.Sin(angleRotated) * distance)
	l.Y = l.Y - (math.Cos(angleRotated) * distance)
}

/**
 * Check if this point is the same as another point
 *
 * @param {Point} point
 * @returns {boolean}
 */
func (l LazyPoint) EqualsTo(p Point.Point) bool {
	return l.X == p.X && l.Y == p.Y
}

/**
 * Get the difference for x and y axis to another point
 *
 * @param {Point} point
 * @returns {Point}
 */
func (l LazyPoint) GetDifferenceTo(p Point.Point) Point.Point {
	newPoint := Point.Point{
		X: l.X - p.X,
		Y: l.Y - p.Y,
	}

	return newPoint
}

/**
 * Calculate distance to another point
 *
 * @param {Point} point
 * @returns {Point}
 */
func (l LazyPoint) GetDistanceTo(p Point.Point) float64 {
	diff := l.GetDifferenceTo(p)

	return math.Sqrt(math.Pow(diff.X, 2) + math.Pow(diff.Y, 2))
}

/**
 * Calculate the angle to another point
 *
 * @param {Point} point
 * @returns {Point}
 */
func (l LazyPoint) GetAngleTo(p Point.Point) float64 {
	diff := l.GetDifferenceTo(p)

	return math.Atan2(diff.Y, diff.X)
}

/**
 * Return a simple object with x and y properties
 *
 * @returns {object}
 */
func (l LazyPoint) ToObject() map[string]float64 {
	return map[string]float64{
		"X": l.X,
		"Y": l.Y,
	}
}
