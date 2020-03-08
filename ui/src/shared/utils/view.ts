// Constants
import {INFERNO, NINETEEN_EIGHTY_FOUR} from '@influxdata/giraffe'
import {DEFAULT_LINE_COLORS} from 'src/shared/constants/graphColorPalettes'
import {DEFAULT_CELL_NAME} from 'src/dashboards/constants/index'
import {
  DEFAULT_GAUGE_COLORS,
  DEFAULT_THRESHOLDS_LIST_COLORS,
} from 'src/shared/constants/thresholds'

// Types
import {ViewType, ViewShape, Base, Scale} from 'src/types'
import {
  XYView,
  XYViewGeom,
  HistogramView,
  HeatmapView,
  LinePlusSingleStatView,
  SingleStatView,
  TableView,
  GaugeView,
  MarkdownView,
  NewView,
  ViewProperties,
  DashboardQuery,
  QueryEditMode,
  BuilderConfig,
  ScatterView,
} from 'src/types/dashboards'

function defaultView() {
  return {
    name: DEFAULT_CELL_NAME,
  }
}

export function defaultViewQuery(): DashboardQuery {
  return {
    name: '',
    text: '',
    editMode: QueryEditMode.Builder,
    builderConfig: defaultBuilderConfig(),
  }
}

export function defaultBuilderConfig(): BuilderConfig {
  return {
    buckets: [],
    tags: [{key: '_measurement', values: []}],
    functions: [],
    aggregateWindow: {period: 'auto'},
  }
}

function defaultLineViewProperties() {
  return {
    queries: [defaultViewQuery()],
    colors: [],
    legend: {},
    note: '',
    showNoteWhenEmpty: false,
    axes: {
      x: {
        bounds: ['', ''] as [string, string],
        label: '',
        prefix: '',
        suffix: '',
        base: '10' as Base,
        scale: Scale.Linear,
      },
      y: {
        bounds: ['', ''] as [string, string],
        label: '',
        prefix: '',
        suffix: '',
        base: '10' as Base,
        scale: Scale.Linear,
      },
    },
  }
}

function defaultGaugeViewProperties() {
  return {
    queries: [defaultViewQuery()],
    colors: DEFAULT_GAUGE_COLORS,
    prefix: '',
    suffix: '',
    note: '',
    showNoteWhenEmpty: false,
    decimalPlaces: {
      isEnforced: true,
      digits: 2,
    },
  }
}

function defaultSingleStatViewProperties() {
  return {
    queries: [defaultViewQuery()],
    colors: DEFAULT_THRESHOLDS_LIST_COLORS,
    prefix: '',
    suffix: '',
    note: '',
    showNoteWhenEmpty: false,
    decimalPlaces: {
      isEnforced: true,
      digits: 2,
    },
  }
}

// Defines the zero values of the various view types
const NEW_VIEW_CREATORS = {
  [ViewType.XY]: (): NewView<XYView> => ({
    ...defaultView(),
    properties: {
      ...defaultLineViewProperties(),
      type: ViewType.XY,
      shape: ViewShape.ChronografV2,
      geom: XYViewGeom.Line,
      xColumn: null,
      yColumn: null,
    },
  }),
  [ViewType.Histogram]: (): NewView<HistogramView> => ({
    ...defaultView(),
    properties: {
      queries: [],
      type: ViewType.Histogram,
      shape: ViewShape.ChronografV2,
      xColumn: '_value',
      xDomain: null,
      xAxisLabel: '',
      fillColumns: null,
      position: 'stacked',
      binCount: 30,
      colors: DEFAULT_LINE_COLORS,
      note: '',
      showNoteWhenEmpty: false,
    },
  }),
  [ViewType.Heatmap]: (): NewView<HeatmapView> => ({
    ...defaultView(),
    properties: {
      queries: [],
      type: ViewType.Heatmap,
      shape: ViewShape.ChronografV2,
      xColumn: null,
      yColumn: null,
      xDomain: null,
      yDomain: null,
      xAxisLabel: '',
      yAxisLabel: '',
      xPrefix: '',
      xSuffix: '',
      yPrefix: '',
      ySuffix: '',
      colors: INFERNO,
      binSize: 10,
      note: '',
      showNoteWhenEmpty: false,
    },
  }),
  [ViewType.SingleStat]: (): NewView<SingleStatView> => ({
    ...defaultView(),
    properties: {
      ...defaultSingleStatViewProperties(),
      type: ViewType.SingleStat,
      shape: ViewShape.ChronografV2,
    },
  }),
  [ViewType.Gauge]: (): NewView<GaugeView> => ({
    ...defaultView(),
    properties: {
      ...defaultGaugeViewProperties(),
      type: ViewType.Gauge,
      shape: ViewShape.ChronografV2,
    },
  }),
  [ViewType.LinePlusSingleStat]: (): NewView<LinePlusSingleStatView> => ({
    ...defaultView(),
    properties: {
      ...defaultLineViewProperties(),
      ...defaultSingleStatViewProperties(),
      type: ViewType.LinePlusSingleStat,
      shape: ViewShape.ChronografV2,
      xColumn: null,
      yColumn: null,
    },
  }),
  [ViewType.Table]: (): NewView<TableView> => ({
    ...defaultView(),
    properties: {
      type: ViewType.Table,
      shape: ViewShape.ChronografV2,
      queries: [defaultViewQuery()],
      colors: DEFAULT_THRESHOLDS_LIST_COLORS,
      tableOptions: {
        verticalTimeAxis: true,
        sortBy: null,
        fixFirstColumn: false,
      },
      fieldOptions: [],
      decimalPlaces: {
        isEnforced: false,
        digits: 2,
      },
      timeFormat: 'YYYY-MM-DD HH:mm:ss',
      note: '',
      showNoteWhenEmpty: false,
    },
  }),
  [ViewType.Markdown]: (): NewView<MarkdownView> => ({
    ...defaultView(),
    properties: {
      type: ViewType.Markdown,
      shape: ViewShape.ChronografV2,
      note: '',
    },
  }),
  [ViewType.Scatter]: (): NewView<ScatterView> => ({
    ...defaultView(),
    properties: {
      type: ViewType.Scatter,
      shape: ViewShape.ChronografV2,
      queries: [defaultViewQuery()],
      colors: NINETEEN_EIGHTY_FOUR,
      note: '',
      showNoteWhenEmpty: false,
      fillColumns: null,
      symbolColumns: null,
      xColumn: null,
      xDomain: null,
      yColumn: null,
      yDomain: null,
      xAxisLabel: '',
      yAxisLabel: '',
      xPrefix: '',
      xSuffix: '',
      yPrefix: '',
      ySuffix: '',
    },
  }),
}

export function createView<T extends ViewProperties = ViewProperties>(
  viewType: ViewType = ViewType.XY
): NewView<T> {
  const creator = NEW_VIEW_CREATORS[viewType]

  if (!creator) {
    throw new Error(`no view creator implemented for view of type ${viewType}`)
  }

  return creator()
}
