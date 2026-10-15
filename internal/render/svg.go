package render

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// SVGs are also copied to the static site, where they can be opened as
// documents. Only a small, non-interactive drawing vocabulary is accepted.
func safeSVG(b []byte) error {
	tags := map[string]bool{"svg": true, "g": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true, "path": true, "text": true, "tspan": true, "defs": true, "linearGradient": true, "radialGradient": true, "stop": true}
	attrs := map[string]bool{"xmlns": true, "width": true, "height": true, "viewBox": true, "role": true, "aria-label": true, "id": true, "class": true, "x": true, "y": true, "x1": true, "y1": true, "x2": true, "y2": true, "cx": true, "cy": true, "r": true, "rx": true, "ry": true, "d": true, "points": true, "fill": true, "fill-opacity": true, "stroke": true, "stroke-width": true, "stroke-opacity": true, "stroke-linecap": true, "stroke-linejoin": true, "opacity": true, "transform": true, "font-family": true, "font-size": true, "font-weight": true, "text-anchor": true, "offset": true, "stop-color": true, "stop-opacity": true, "gradientUnits": true}
	d := xml.NewDecoder(bytes.NewReader(b))
	depth, roots := 0, 0
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid SVG: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !tags[t.Name.Local] || (t.Name.Space != "" && t.Name.Space != "http://www.w3.org/2000/svg") {
				return fmt.Errorf("unsafe SVG element %s", t.Name.Local)
			}
			if depth == 0 {
				roots++
				if t.Name.Local != "svg" {
					return errors.New("SVG root required")
				}
			}
			depth++
			for _, a := range t.Attr {
				if (a.Name.Space != "" && a.Name.Space != "xmlns") || !attrs[a.Name.Local] || strings.Contains(strings.ToLower(a.Value), "url(") {
					return fmt.Errorf("unsafe SVG attribute %s", a.Name.Local)
				}
			}
		case xml.EndElement:
			depth--
		case xml.ProcInst:
			if t.Target != "xml" || depth != 0 || roots != 0 {
				return errors.New("unsafe SVG processing instruction")
			}
		case xml.Directive:
			return errors.New("unsafe SVG directive")
		}
	}
	if roots != 1 || depth != 0 {
		return errors.New("invalid SVG document")
	}
	return nil
}
