package draw

import (
	"image"
	"image/color"
	"sort"
)

func FillPolygon(dst *image.RGBA, points []Point, c color.RGBA) {
	if len(points) < 3 {
		return // Not a polygon
	}

	// Find min and max Y
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Iterate through each scanline
	for y := minY; y <= maxY; y++ {
		var nodeX []float64

		// Build a list of node intersections
		j := len(points) - 1
		for i := 0; i < len(points); i++ {
			if (points[i].Y < y && points[j].Y >= y) || (points[j].Y < y && points[i].Y >= y) {
				nodeX = append(nodeX, points[i].X+(y-points[i].Y)*(points[j].X-points[i].X)/(points[j].Y-points[i].Y))
			}
			j = i
		}

		// Sort the nodes
		sort.Float64s(nodeX)

		// Fill the pixels between pairs of nodes
		for i := 0; i < len(nodeX); i += 2 {
			if nodeX[i] >= float64(dst.Bounds().Max.X) || nodeX[i+1] < 0 {
				continue
			}
			if nodeX[i] < 0 {
				nodeX[i] = 0
			}
			if nodeX[i+1] >= float64(dst.Bounds().Max.X) {
				nodeX[i+1] = float64(dst.Bounds().Max.X) - 1
			}
			for x := nodeX[i]; x < nodeX[i+1]; x++ {
				dst.SetRGBA(int(x), int(y), c)
			}
		}
	}

	// Anti-aliasing edges
	for i := 0; i < len(points); i++ {
		nextIndex := (i + 1) % len(points)
		_ = nextIndex
		drawWuLine(dst, points[i].X, points[i].Y, points[nextIndex].X, points[nextIndex].Y, c)
	}
}
