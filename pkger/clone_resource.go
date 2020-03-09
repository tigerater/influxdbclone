package pkger

import (
	"errors"
	"sort"

	"github.com/influxdata/influxdb"
)

// ResourceToClone is a resource that will be cloned.
type ResourceToClone struct {
	Kind Kind        `json:"kind"`
	ID   influxdb.ID `json:"id"`
	Name string      `json:"name"`
}

// OK validates a resource clone is viable.
func (r ResourceToClone) OK() error {
	if err := r.Kind.OK(); err != nil {
		return err
	}
	if r.ID == influxdb.ID(0) {
		return errors.New("must provide an ID")
	}
	return nil
}

func uniqResourcesToClone(resources []ResourceToClone) []ResourceToClone {
	type key struct {
		kind Kind
		id   influxdb.ID
	}
	m := make(map[key]ResourceToClone)

	for i := range resources {
		r := resources[i]
		rKey := key{kind: r.Kind, id: r.ID}
		kr, ok := m[rKey]
		switch {
		case ok && kr.Name == r.Name:
		case ok && kr.Name != "" && r.Name == "":
		default:
			m[rKey] = r
		}
	}

	out := make([]ResourceToClone, 0, len(resources))
	for _, r := range m {
		out = append(out, r)
	}
	return out
}

func bucketToResource(bkt influxdb.Bucket, name string) Resource {
	if name == "" {
		name = bkt.Name
	}
	r := Resource{
		fieldKind: KindBucket.title(),
		fieldName: name,
	}
	assignNonZeroStrings(r, map[string]string{fieldDescription: bkt.Description})
	if bkt.RetentionPeriod != 0 {
		r[fieldBucketRetentionRules] = retentionRules{newRetentionRule(bkt.RetentionPeriod)}
	}
	return r
}

type cellView struct {
	c influxdb.Cell
	v influxdb.View
}

func convertCellView(cv cellView) chart {
	ch := chart{
		Name:   cv.v.Name,
		Height: int(cv.c.H),
		Width:  int(cv.c.W),
		XPos:   int(cv.c.X),
		YPos:   int(cv.c.Y),
	}

	setCommon := func(k chartKind, iColors []influxdb.ViewColor, dec influxdb.DecimalPlaces, iQueries []influxdb.DashboardQuery) {
		ch.Kind = k
		ch.Colors = convertColors(iColors)
		ch.DecimalPlaces = int(dec.Digits)
		ch.EnforceDecimals = dec.IsEnforced
		ch.Queries = convertQueries(iQueries)
	}

	setNoteFixes := func(note string, noteOnEmpty bool, prefix, suffix string) {
		ch.Note = note
		ch.NoteOnEmpty = noteOnEmpty
		ch.Prefix = prefix
		ch.Suffix = suffix
	}

	setLegend := func(l influxdb.Legend) {
		ch.Legend.Orientation = l.Orientation
		ch.Legend.Type = l.Type
	}

	props := cv.v.Properties
	switch p := props.(type) {
	case influxdb.GaugeViewProperties:
		setCommon(chartKindGauge, p.ViewColors, p.DecimalPlaces, p.Queries)
		setNoteFixes(p.Note, p.ShowNoteWhenEmpty, p.Prefix, p.Suffix)
	case influxdb.HeatmapViewProperties:
		ch.Kind = chartKindHeatMap
		ch.Queries = convertQueries(p.Queries)
		ch.Colors = stringsToColors(p.ViewColors)
		ch.XCol = p.XColumn
		ch.YCol = p.YColumn
		ch.Axes = []axis{
			{Label: p.XAxisLabel, Prefix: p.XPrefix, Suffix: p.XSuffix, Name: "x", Domain: p.XDomain},
			{Label: p.YAxisLabel, Prefix: p.YPrefix, Suffix: p.YSuffix, Name: "y", Domain: p.YDomain},
		}
		ch.Note = p.Note
		ch.NoteOnEmpty = p.ShowNoteWhenEmpty
		ch.BinSize = int(p.BinSize)
	case influxdb.HistogramViewProperties:
		ch.Kind = chartKindHistogram
		ch.Queries = convertQueries(p.Queries)
		ch.Colors = convertColors(p.ViewColors)
		ch.XCol = p.XColumn
		ch.Axes = []axis{{Label: p.XAxisLabel, Name: "x", Domain: p.XDomain}}
		ch.Note = p.Note
		ch.NoteOnEmpty = p.ShowNoteWhenEmpty
		ch.BinCount = p.BinCount
		ch.Position = p.Position
	case influxdb.MarkdownViewProperties:
		ch.Kind = chartKindMarkdown
		ch.Note = p.Note
	case influxdb.LinePlusSingleStatProperties:
		setCommon(chartKindSingleStatPlusLine, p.ViewColors, p.DecimalPlaces, p.Queries)
		setNoteFixes(p.Note, p.ShowNoteWhenEmpty, p.Prefix, p.Suffix)
		setLegend(p.Legend)
		ch.Axes = convertAxes(p.Axes)
		ch.Shade = p.ShadeBelow
		ch.XCol = p.XColumn
		ch.YCol = p.YColumn
		ch.Position = p.Position
	case influxdb.SingleStatViewProperties:
		setCommon(chartKindSingleStat, p.ViewColors, p.DecimalPlaces, p.Queries)
		setNoteFixes(p.Note, p.ShowNoteWhenEmpty, p.Prefix, p.Suffix)
	case influxdb.ScatterViewProperties:
		ch.Kind = chartKindScatter
		ch.Queries = convertQueries(p.Queries)
		ch.Colors = stringsToColors(p.ViewColors)
		ch.XCol = p.XColumn
		ch.YCol = p.YColumn
		ch.Axes = []axis{
			{Label: p.XAxisLabel, Prefix: p.XPrefix, Suffix: p.XSuffix, Name: "x", Domain: p.XDomain},
			{Label: p.YAxisLabel, Prefix: p.YPrefix, Suffix: p.YSuffix, Name: "y", Domain: p.YDomain},
		}
		ch.Note = p.Note
		ch.NoteOnEmpty = p.ShowNoteWhenEmpty
	case influxdb.XYViewProperties:
		setCommon(chartKindXY, p.ViewColors, influxdb.DecimalPlaces{}, p.Queries)
		setNoteFixes(p.Note, p.ShowNoteWhenEmpty, "", "")
		setLegend(p.Legend)
		ch.Axes = convertAxes(p.Axes)
		ch.Geom = p.Geom
		ch.Shade = p.ShadeBelow
		ch.XCol = p.XColumn
		ch.YCol = p.YColumn
		ch.Position = p.Position
	}

	return ch
}

func convertChartToResource(ch chart) Resource {
	r := Resource{
		fieldKind:         ch.Kind.title(),
		fieldName:         ch.Name,
		fieldChartQueries: ch.Queries,
		fieldChartHeight:  ch.Height,
		fieldChartWidth:   ch.Width,
	}
	if len(ch.Colors) > 0 {
		r[fieldChartColors] = ch.Colors
	}
	if len(ch.Axes) > 0 {
		r[fieldChartAxes] = ch.Axes
	}
	if ch.EnforceDecimals {
		r[fieldChartDecimalPlaces] = ch.DecimalPlaces
	}

	if ch.Legend.Type != "" {
		r[fieldChartLegend] = ch.Legend
	}

	assignNonZeroBools(r, map[string]bool{
		fieldChartNoteOnEmpty: ch.NoteOnEmpty,
		fieldChartShade:       ch.Shade,
	})

	assignNonZeroStrings(r, map[string]string{
		fieldChartNote:     ch.Note,
		fieldPrefix:        ch.Prefix,
		fieldSuffix:        ch.Suffix,
		fieldChartGeom:     ch.Geom,
		fieldChartXCol:     ch.XCol,
		fieldChartYCol:     ch.YCol,
		fieldChartPosition: ch.Position,
	})

	assignNonZeroInts(r, map[string]int{
		fieldChartXPos:     ch.XPos,
		fieldChartYPos:     ch.YPos,
		fieldChartBinCount: ch.BinCount,
		fieldChartBinSize:  ch.BinSize,
	})

	return r
}

func convertAxes(iAxes map[string]influxdb.Axis) axes {
	out := make(axes, 0, len(iAxes))
	for name, a := range iAxes {
		out = append(out, axis{
			Base:   a.Base,
			Label:  a.Label,
			Name:   name,
			Prefix: a.Prefix,
			Scale:  a.Scale,
			Suffix: a.Suffix,
		})
	}
	return out
}

func convertColors(iColors []influxdb.ViewColor) colors {
	out := make(colors, 0, len(iColors))
	for _, ic := range iColors {
		out = append(out, &color{
			Name:  ic.Name,
			Type:  ic.Type,
			Hex:   ic.Hex,
			Value: flt64Ptr(ic.Value),
		})
	}
	return out
}

func convertQueries(iQueries []influxdb.DashboardQuery) queries {
	out := make(queries, 0, len(iQueries))
	for _, iq := range iQueries {
		out = append(out, query{Query: iq.Text})
	}
	return out
}

func dashboardToResource(dash influxdb.Dashboard, cellViews []cellView, name string) Resource {
	if name == "" {
		name = dash.Name
	}

	sort.Slice(cellViews, func(i, j int) bool {
		ic, jc := cellViews[i].c, cellViews[j].c
		if ic.X == jc.X {
			return ic.Y < jc.Y
		}
		return ic.X < jc.X
	})

	charts := make([]Resource, 0, len(cellViews))
	for _, cv := range cellViews {
		if cv.c.ID == influxdb.ID(0) {
			continue
		}
		ch := convertCellView(cv)
		if !ch.Kind.ok() {
			continue
		}
		charts = append(charts, convertChartToResource(ch))
	}

	return Resource{
		fieldKind:        KindDashboard.title(),
		fieldName:        name,
		fieldDescription: dash.Description,
		fieldDashCharts:  charts,
	}
}

func labelToResource(l influxdb.Label, name string) Resource {
	if name == "" {
		name = l.Name
	}
	r := Resource{
		fieldKind: KindLabel.title(),
		fieldName: name,
	}

	assignNonZeroStrings(r, map[string]string{
		fieldDescription: l.Properties["description"],
		fieldLabelColor:  l.Properties["color"],
	})
	return r
}

func telegrafToResource(t influxdb.TelegrafConfig, name string) Resource {
	if name == "" {
		name = t.Name
	}
	r := Resource{
		fieldKind:           KindTelegraf.title(),
		fieldName:           name,
		fieldTelegrafConfig: t.TOML(),
	}
	assignNonZeroStrings(r, map[string]string{
		fieldDescription: t.Description,
	})
	return r
}

func variableToResource(v influxdb.Variable, name string) Resource {
	if name == "" {
		name = v.Name
	}

	r := Resource{
		fieldKind: KindVariable.title(),
		fieldName: name,
	}
	assignNonZeroStrings(r, map[string]string{fieldDescription: v.Description})

	args := v.Arguments
	if args == nil {
		return r
	}
	r[fieldType] = args.Type

	switch args.Type {
	case fieldArgTypeConstant:
		vals, ok := args.Values.(influxdb.VariableConstantValues)
		if ok {
			r[fieldValues] = []string(vals)
		}
	case fieldArgTypeMap:
		vals, ok := args.Values.(influxdb.VariableMapValues)
		if ok {
			r[fieldValues] = map[string]string(vals)
		}
	case fieldArgTypeQuery:
		vals, ok := args.Values.(influxdb.VariableQueryValues)
		if ok {
			r[fieldLanguage] = vals.Language
			r[fieldQuery] = vals.Query
		}
	}

	return r
}

func assignNonZeroBools(r Resource, m map[string]bool) {
	for k, v := range m {
		if v {
			r[k] = v
		}
	}
}

func assignNonZeroInts(r Resource, m map[string]int) {
	for k, v := range m {
		if v != 0 {
			r[k] = v
		}
	}
}

func assignNonZeroStrings(r Resource, m map[string]string) {
	for k, v := range m {
		if v != "" {
			r[k] = v
		}
	}
}

func stringsToColors(clrs []string) colors {
	newColors := make(colors, 0)
	for _, x := range clrs {
		newColors = append(newColors, &color{Hex: x})
	}
	return newColors
}
