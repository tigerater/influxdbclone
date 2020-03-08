// Libraries
import React, {PureComponent} from 'react'

// Components
import {Dropdown} from '@influxdata/clockface'

// Types
import {WritePrecision} from '@influxdata/influx'
import {Precision} from 'src/types/dataLoaders'

interface Props {
  setPrecision: (precision: WritePrecision) => void
  precision: WritePrecision
}

const writePrecisions: WritePrecision[] = [
  WritePrecision.Ns,
  WritePrecision.Us,
  WritePrecision.Ms,
  WritePrecision.S,
]

const makePrecisionReadable = {
  [WritePrecision.Ns]: Precision.Nanoseconds,
  [WritePrecision.Us]: Precision.Microseconds,
  [WritePrecision.S]: Precision.Seconds,
  [WritePrecision.Ms]: Precision.Milliseconds,
}

class PrecisionDropdown extends PureComponent<Props> {
  public render() {
    const {setPrecision, precision} = this.props
    return (
      <div className="wizard-step--lp-precision">
        <label>Time Precision</label>
        <Dropdown
          widthPixels={200}
          className="wizard-step--lp-precision"
          button={(active, onClick) => (
            <Dropdown.Button active={active} onClick={onClick}>
              {makePrecisionReadable[precision]}
            </Dropdown.Button>
          )}
          menu={onCollapse => (
            <Dropdown.Menu onCollapse={onCollapse}>
              {writePrecisions.map(value => (
                <Dropdown.Item
                  key={value}
                  value={value}
                  id={value}
                  onClick={setPrecision}
                  selected={`${value}` === `${precision}`}
                >
                  {makePrecisionReadable[value]}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          )}
        />
      </div>
    )
  }
}

export default PrecisionDropdown
