// Libraries
import React, {PureComponent, ChangeEvent, FormEvent} from 'react'
import moment from 'moment'

// Components
import {
  Form,
  Input,
  Button,
  ComponentColor,
  ComponentStatus,
  ButtonType,
  Grid,
} from 'src/clockface'
import Retention from 'src/organizations/components/Retention'

// Constants
import {MIN_RETENTION_SECONDS} from 'src/organizations/constants'

// Types
import {BucketRetentionRules} from '@influxdata/influx'

interface Props {
  name: string
  nameErrorMessage: string
  retentionSeconds: number
  ruleType: BucketRetentionRules.TypeEnum
  onSubmit: (e: FormEvent<HTMLFormElement>) => void
  onCloseModal: () => void
  onChangeRetentionRule: (seconds: number) => void
  onChangeRuleType: (t: BucketRetentionRules.TypeEnum) => void
  onChangeInput: (e: ChangeEvent<HTMLInputElement>) => void
  nameInputStatus: ComponentStatus
  buttonText: string
}

export default class BucketOverlayForm extends PureComponent<Props> {
  public render() {
    const {
      name,
      onSubmit,
      ruleType,
      buttonText,
      nameErrorMessage,
      retentionSeconds,
      nameInputStatus,
      onCloseModal,
      onChangeInput,
      onChangeRuleType,
      onChangeRetentionRule,
    } = this.props

    return (
      <Form onSubmit={onSubmit}>
        <Grid>
          <Grid.Row>
            <Grid.Column>
              <Form.Element label="Name" errorMessage={nameErrorMessage}>
                <Input
                  placeholder="Give your bucket a name"
                  name="name"
                  autoFocus={true}
                  value={name}
                  onChange={onChangeInput}
                  status={nameInputStatus}
                />
              </Form.Element>
              <Form.Element
                label="How often to clear data?"
                errorMessage={this.ruleErrorMessage}
              >
                <Retention
                  type={ruleType}
                  retentionSeconds={retentionSeconds}
                  onChangeRuleType={onChangeRuleType}
                  onChangeRetentionRule={onChangeRetentionRule}
                />
              </Form.Element>
            </Grid.Column>
          </Grid.Row>
          <Grid.Row>
            <Grid.Column>
              <Form.Footer>
                <Button
                  text="Cancel"
                  onClick={onCloseModal}
                  type={ButtonType.Button}
                />
                <Button
                  text={buttonText}
                  color={this.submitButtonColor}
                  status={this.submitButtonStatus}
                  type={ButtonType.Submit}
                />
              </Form.Footer>
            </Grid.Column>
          </Grid.Row>
        </Grid>
      </Form>
    )
  }

  private get submitButtonColor(): ComponentColor {
    const {buttonText} = this.props

    if (buttonText === 'Save Changes') {
      return ComponentColor.Success
    }

    return ComponentColor.Primary
  }

  private get submitButtonStatus(): ComponentStatus {
    const {name} = this.props

    const nameEmpty = name === ''

    if (nameEmpty || this.retentionIsTooShort) {
      return ComponentStatus.Disabled
    }

    return ComponentStatus.Default
  }

  private get retentionIsTooShort(): boolean {
    const {retentionSeconds, ruleType} = this.props

    return (
      ruleType === BucketRetentionRules.TypeEnum.Expire &&
      retentionSeconds < MIN_RETENTION_SECONDS
    )
  }

  private get ruleErrorMessage(): string {
    if (this.retentionIsTooShort) {
      const humanDuration = moment
        .duration(MIN_RETENTION_SECONDS, 'seconds')
        .humanize()

      return `Retention period must be at least ${humanDuration}`
    }

    return ''
  }
}
