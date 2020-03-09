// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'

// Components
import {Grid} from '@influxdata/clockface'
import Geom from 'src/timeMachine/components/view_options/Geom'
import YAxisTitle from 'src/timeMachine/components/view_options/YAxisTitle'
import AxisAffixes from 'src/timeMachine/components/view_options/AxisAffixes'
import ColorSelector from 'src/timeMachine/components/view_options/ColorSelector'
import AutoDomainInput from 'src/shared/components/AutoDomainInput'
import YAxisBase from 'src/timeMachine/components/view_options/YAxisBase'
import ColumnSelector from 'src/shared/components/ColumnSelector'
import Checkbox from 'src/shared/components/Checkbox'

// Actions
import {
  setColors,
  setYAxisLabel,
  setAxisPrefix,
  setAxisSuffix,
  setYAxisBounds,
  setYAxisBase,
  setGeom,
  setXColumn,
  setYColumn,
  setShadeBelow,
} from 'src/timeMachine/actions'

// Utils
import {parseBounds} from 'src/shared/utils/vis'
import {
  getXColumnSelection,
  getYColumnSelection,
  getNumericColumns,
} from 'src/timeMachine/selectors'

// Types
import {ViewType} from 'src/types'
import {AppState, XYGeom, Axes, Color} from 'src/types'

interface OwnProps {
  type: ViewType
  axes: Axes
  geom?: XYGeom
  colors: Color[]
  shadeBelow?: boolean
}

interface StateProps {
  xColumn: string
  yColumn: string
  numericColumns: string[]
}

interface DispatchProps {
  onUpdateYAxisLabel: typeof setYAxisLabel
  onUpdateAxisPrefix: typeof setAxisPrefix
  onUpdateAxisSuffix: typeof setAxisSuffix
  onUpdateYAxisBounds: typeof setYAxisBounds
  onUpdateYAxisBase: typeof setYAxisBase
  onUpdateColors: typeof setColors
  onSetShadeBelow: typeof setShadeBelow
  onSetXColumn: typeof setXColumn
  onSetYColumn: typeof setYColumn
  onSetGeom: typeof setGeom
}

type Props = OwnProps & DispatchProps & StateProps

class LineOptions extends PureComponent<Props> {
  public render() {
    const {
      axes: {
        y: {label, prefix, suffix, base},
      },
      colors,
      geom,
      shadeBelow,
      onUpdateColors,
      onUpdateYAxisLabel,
      onUpdateAxisPrefix,
      onUpdateAxisSuffix,
      onUpdateYAxisBase,
      onSetShadeBelow,
      onSetGeom,
      onSetYColumn,
      yColumn,
      onSetXColumn,
      xColumn,
      numericColumns,
    } = this.props

    return (
      <>
        <Grid.Column>
          <h4 className="view-options--header">Customize Line Graph</h4>
          <h5 className="view-options--header">Data</h5>
          <ColumnSelector
            selectedColumn={xColumn}
            onSelectColumn={onSetXColumn}
            availableColumns={numericColumns}
            axisName="x"
          />
          <ColumnSelector
            selectedColumn={yColumn}
            onSelectColumn={onSetYColumn}
            availableColumns={numericColumns}
            axisName="y"
          />
          <h5 className="view-options--header">Options</h5>
        </Grid.Column>
        {geom && <Geom geom={geom} onSetGeom={onSetGeom} />}
        <ColorSelector
          colors={colors.filter(c => c.type === 'scale')}
          onUpdateColors={onUpdateColors}
        />
        <Grid.Column>
          <Checkbox
            label="Shade Area Below Lines"
            checked={!!shadeBelow}
            onSetChecked={onSetShadeBelow}
          />
        </Grid.Column>
        <Grid.Column>
          <h5 className="view-options--header">Y Axis</h5>
        </Grid.Column>
        <YAxisTitle label={label} onUpdateYAxisLabel={onUpdateYAxisLabel} />
        <YAxisBase base={base} onUpdateYAxisBase={onUpdateYAxisBase} />
        <AxisAffixes
          prefix={prefix}
          suffix={suffix}
          axisName="y"
          onUpdateAxisPrefix={prefix => onUpdateAxisPrefix(prefix, 'y')}
          onUpdateAxisSuffix={suffix => onUpdateAxisSuffix(suffix, 'y')}
        />
        <Grid.Column>
          <AutoDomainInput
            domain={this.yDomain}
            onSetDomain={this.handleSetYDomain}
            label="Y Axis Domain"
          />
        </Grid.Column>
      </>
    )
  }

  private get yDomain(): [number, number] {
    return parseBounds(this.props.axes.y.bounds)
  }

  private handleSetYDomain = (yDomain: [number, number]): void => {
    let bounds: [string, string] | [null, null]

    if (yDomain) {
      bounds = [String(yDomain[0]), String(yDomain[1])]
    } else {
      bounds = [null, null]
    }

    this.props.onUpdateYAxisBounds(bounds)
  }
}

const mstp = (state: AppState) => {
  const xColumn = getXColumnSelection(state)
  const yColumn = getYColumnSelection(state)
  const numericColumns = getNumericColumns(state)

  return {xColumn, yColumn, numericColumns}
}

const mdtp: DispatchProps = {
  onUpdateYAxisLabel: setYAxisLabel,
  onUpdateAxisPrefix: setAxisPrefix,
  onUpdateAxisSuffix: setAxisSuffix,
  onUpdateYAxisBounds: setYAxisBounds,
  onUpdateYAxisBase: setYAxisBase,
  onSetXColumn: setXColumn,
  onSetYColumn: setYColumn,
  onSetShadeBelow: setShadeBelow,
  onUpdateColors: setColors,
  onSetGeom: setGeom,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(LineOptions)
