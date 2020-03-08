// Libraries
import React, {FC, ChangeEvent, useContext} from 'react'

// Components
import {
  Radio,
  Form,
  Input,
  InputType,
  Grid,
  Columns,
  ButtonShape,
} from '@influxdata/clockface'
import {NewRuleDispatch} from './NewRuleOverlay'

// Types
import {RuleState} from './NewRuleOverlay.reducer'

interface Props {
  rule: RuleState
  onChange: (e: ChangeEvent) => void
}

const RuleSchedule: FC<Props> = ({rule, onChange}) => {
  const {schedule, cron, every, offset} = rule
  const label = schedule === 'every' ? 'Every' : 'Cron'
  const placeholder = schedule === 'every' ? '1d3h30s' : '0 2 * * *'
  const value = schedule === 'every' ? every : cron
  const dispatch = useContext(NewRuleDispatch)

  return (
    <Grid.Row>
      <Grid.Column widthXS={Columns.Four}>
        <Form.Element label="Schedule">
          <Radio shape={ButtonShape.StretchToFit}>
            <Radio.Button
              id="every"
              testID="rule-schedule-every"
              active={schedule === 'every'}
              value="every"
              titleText="Run task at regular intervals"
              onClick={() =>
                dispatch({
                  type: 'SET_ACTIVE_SCHEDULE',
                  schedule: 'every',
                })
              }
            >
              Every
            </Radio.Button>
            <Radio.Button
              id="cron"
              testID="rule-schedule-cron"
              active={schedule === 'cron'}
              value="cron"
              titleText="Use cron syntax for more control over scheduling"
              onClick={() =>
                dispatch({
                  type: 'SET_ACTIVE_SCHEDULE',
                  schedule: 'cron',
                })
              }
            >
              Cron
            </Radio.Button>
          </Radio>
        </Form.Element>
      </Grid.Column>
      <Grid.Column widthXS={Columns.Four}>
        <Form.Element label={label}>
          <Input
            value={value}
            name={schedule}
            type={InputType.Text}
            placeholder={placeholder}
            onChange={onChange}
            testID={`rule-schedule-${schedule}--input`}
          />
        </Form.Element>
      </Grid.Column>

      <Grid.Column widthXS={Columns.Four}>
        <Form.Element label="Offset">
          <Input
            name="offset"
            type={InputType.Text}
            value={offset}
            placeholder="20m"
            onChange={onChange}
            testID="rule-schedule-offset--input"
          />
        </Form.Element>
      </Grid.Column>
    </Grid.Row>
  )
}

export default RuleSchedule
