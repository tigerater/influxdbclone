// Libraries
import React, {PureComponent, ChangeEvent, FormEvent} from 'react'

// Components
import {Overlay, ComponentStatus} from '@influxdata/clockface'
import BucketOverlayForm from 'src/buckets/components/BucketOverlayForm'

// Types
import {Organization, Bucket} from 'src/types'

interface Props {
  org: Organization
  onCloseModal: () => void
  onCreateBucket: (bucket: Partial<Bucket>) => Promise<void>
}

interface State {
  bucket: Bucket
  ruleType: 'expire'
  nameInputStatus: ComponentStatus
  nameErrorMessage: string
}

const emptyBucket = {
  name: '',
  retentionRules: [],
} as Bucket

export default class CreateBucketOverlay extends PureComponent<Props, State> {
  constructor(props) {
    super(props)
    this.state = {
      nameErrorMessage: '',
      bucket: emptyBucket,
      ruleType: null,
      nameInputStatus: ComponentStatus.Default,
    }
  }

  public render() {
    const {onCloseModal} = this.props
    const {bucket, nameInputStatus, nameErrorMessage, ruleType} = this.state

    return (
      <Overlay.Container maxWidth={500}>
        <Overlay.Header
          title="Create Bucket"
          onDismiss={this.props.onCloseModal}
        />
        <Overlay.Body>
          <BucketOverlayForm
            name={bucket.name}
            buttonText="Create"
            ruleType={ruleType}
            onCloseModal={onCloseModal}
            nameErrorMessage={nameErrorMessage}
            onSubmit={this.handleSubmit}
            nameInputStatus={nameInputStatus}
            onChangeInput={this.handleChangeInput}
            retentionSeconds={this.retentionSeconds}
            onChangeRuleType={this.handleChangeRuleType}
            onChangeRetentionRule={this.handleChangeRetentionRule}
          />
        </Overlay.Body>
      </Overlay.Container>
    )
  }

  private get retentionSeconds(): number {
    const rule = this.state.bucket.retentionRules.find(r => r.type === 'expire')

    if (!rule) {
      return 3600
    }

    return rule.everySeconds
  }

  private handleChangeRetentionRule = (everySeconds: number): void => {
    const bucket = {
      ...this.state.bucket,
      retentionRules: [{type: 'expire' as 'expire', everySeconds}],
    }

    this.setState({bucket})
  }

  private handleChangeRuleType = ruleType => {
    this.setState({ruleType})
  }

  private handleSubmit = (e: FormEvent<HTMLFormElement>): void => {
    e.preventDefault()
    this.handleCreateBucket()
  }

  private handleCreateBucket = (): void => {
    const {onCreateBucket, org} = this.props
    const orgID = org.id
    const organization = org.name

    const bucket = {
      ...this.state.bucket,
      orgID,
      organization,
    }

    onCreateBucket(bucket)
  }

  private handleChangeInput = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    const key = e.target.name
    const bucket = {...this.state.bucket, [key]: value}

    if (!value) {
      return this.setState({
        bucket,
        nameInputStatus: ComponentStatus.Error,
        nameErrorMessage: `Bucket ${key} cannot be empty`,
      })
    }

    this.setState({
      bucket,
      nameInputStatus: ComponentStatus.Valid,
      nameErrorMessage: '',
    })
  }
}
