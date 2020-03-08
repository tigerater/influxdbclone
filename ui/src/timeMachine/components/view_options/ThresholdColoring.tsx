// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import {get} from 'lodash'

// Components
import {Form, Radio, ButtonShape} from 'src/clockface'

// Actions
import {
  setBackgroundThresholdColoring,
  setTextThresholdColoring,
} from 'src/timeMachine/actions'

// Utils
import {getActiveTimeMachine} from 'src/timeMachine/selectors'

// Types
import {AppState} from 'src/types/v2'
import {Color} from 'src/types/colors'

enum ThresholdColoringSetting {
  Background = 'background',
  Text = 'text',
}

interface StateProps {
  colors: Color[]
}

interface DispatchProps {
  onSetBackground: typeof setBackgroundThresholdColoring
  onSetText: typeof setTextThresholdColoring
}

type Props = StateProps & DispatchProps

class ThresholdColoring extends PureComponent<Props> {
  public render() {
    return (
      <Form.Element label="Colorization">
        <Radio shape={ButtonShape.StretchToFit}>
          <Radio.Button
            active={this.activeSetting === ThresholdColoringSetting.Background}
            onClick={this.handleClick}
            value={ThresholdColoringSetting.Background}
          >
            Background
          </Radio.Button>
          <Radio.Button
            active={this.activeSetting === ThresholdColoringSetting.Text}
            onClick={this.handleClick}
            value={ThresholdColoringSetting.Text}
          >
            Text
          </Radio.Button>
        </Radio>
      </Form.Element>
    )
  }

  private get activeSetting(): ThresholdColoringSetting {
    const {colors} = this.props
    const activeSetting: ThresholdColoringSetting = get(
      colors.filter(c => c.type !== 'scale'),
      '0.type',
      ThresholdColoringSetting.Text
    )

    return activeSetting
  }

  private handleClick = (setting: ThresholdColoringSetting): void => {
    if (setting === ThresholdColoringSetting.Background) {
      this.props.onSetBackground()
    } else if (setting === ThresholdColoringSetting.Text) {
      this.props.onSetText()
    }
  }
}

const mstp = (state: AppState) => {
  const colors = getActiveTimeMachine(state).view.properties.colors

  return {colors}
}

const mdtp = {
  onSetBackground: setBackgroundThresholdColoring,
  onSetText: setTextThresholdColoring,
}

export default connect<StateProps, DispatchProps>(
  mstp,
  mdtp
)(ThresholdColoring)
