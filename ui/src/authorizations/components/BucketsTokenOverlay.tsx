import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {Form} from 'src/clockface'
import {
  IconFont,
  ComponentColor,
  ComponentSpacer,
  AlignItems,
  FlexDirection,
  ComponentSize,
  Button,
  ButtonType,
  Grid,
  Columns,
  Input,
  Overlay,
} from '@influxdata/clockface'
import BucketsSelector from 'src/authorizations/components/BucketsSelector'
import GetResources, {ResourceTypes} from 'src/shared/components/GetResources'

// Utils
import {
  specificBucketsPermissions,
  selectBucket,
  allBucketsPermissions,
  BucketTab,
} from 'src/authorizations/utils/permissions'

// Actions
import {createAuthorization} from 'src/authorizations/actions'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

// Types
import {AppState, Bucket} from 'src/types'
import {Permission, Authorization} from '@influxdata/influx'

interface StateProps {
  buckets: Bucket[]
}

interface DispatchProps {
  onCreateAuthorization: typeof createAuthorization
}

interface State {
  description: string
  readBuckets: string[]
  writeBuckets: string[]
  activeTabRead: BucketTab
  activeTabWrite: BucketTab
}

type Props = WithRouterProps & DispatchProps & StateProps

@ErrorHandling
class BucketsTokenOverlay extends PureComponent<Props, State> {
  public state = {
    description: '',
    readBuckets: [],
    writeBuckets: [],
    activeTabRead: BucketTab.Scoped,
    activeTabWrite: BucketTab.Scoped,
  }

  render() {
    const {buckets} = this.props
    const {
      description,
      readBuckets,
      writeBuckets,
      activeTabRead,
      activeTabWrite,
    } = this.state

    return (
      <Overlay visible={true}>
        <Overlay.Container>
          <Overlay.Header
            title="Generate Read/Write Token"
            onDismiss={this.handleDismiss}
          />
          <Overlay.Body>
            <Form onSubmit={this.handleSave}>
              <ComponentSpacer
                alignItems={AlignItems.Center}
                direction={FlexDirection.Column}
                margin={ComponentSize.Large}
              >
                <Form.Element label="Description">
                  <Input
                    placeholder="Describe this new token"
                    value={description}
                    onChange={this.handleInputChange}
                    testID="input-field--descr"
                  />
                </Form.Element>
                <Form.Element label="">
                  <GetResources resource={ResourceTypes.Buckets}>
                    <Grid.Row>
                      <Grid.Column
                        widthXS={Columns.Twelve}
                        widthSM={Columns.Six}
                      >
                        <BucketsSelector
                          onSelect={this.handleSelectReadBucket}
                          buckets={buckets}
                          selectedBuckets={readBuckets}
                          title="Read"
                          onSelectAll={this.handleReadSelectAllBuckets}
                          onDeselectAll={this.handleReadDeselectAllBuckets}
                          activeTab={activeTabRead}
                          onTabClick={this.handleReadTabClick}
                        />
                      </Grid.Column>
                      <Grid.Column
                        widthXS={Columns.Twelve}
                        widthSM={Columns.Six}
                      >
                        <BucketsSelector
                          onSelect={this.handleSelectWriteBucket}
                          buckets={buckets}
                          selectedBuckets={writeBuckets}
                          title="Write"
                          onSelectAll={this.handleWriteSelectAllBuckets}
                          onDeselectAll={this.handleWriteDeselectAllBuckets}
                          activeTab={activeTabWrite}
                          onTabClick={this.handleWriteTabClick}
                        />
                      </Grid.Column>
                    </Grid.Row>
                  </GetResources>
                </Form.Element>
                <ComponentSpacer
                  alignItems={AlignItems.Center}
                  direction={FlexDirection.Row}
                  margin={ComponentSize.Small}
                >
                  <Button
                    text="Cancel"
                    icon={IconFont.Remove}
                    onClick={this.handleDismiss}
                    testID="button--cancel"
                  />

                  <Button
                    text="Save"
                    icon={IconFont.Checkmark}
                    color={ComponentColor.Success}
                    type={ButtonType.Submit}
                    testID="button--save"
                  />
                </ComponentSpacer>
              </ComponentSpacer>
            </Form>
          </Overlay.Body>
        </Overlay.Container>
      </Overlay>
    )
  }

  private handleReadTabClick = (tab: BucketTab) => {
    this.setState({activeTabRead: tab})
  }

  private handleWriteTabClick = (tab: BucketTab) => {
    this.setState({activeTabWrite: tab})
  }

  private handleSelectReadBucket = (bucketName: string): void => {
    const readBuckets = selectBucket(bucketName, this.state.readBuckets)

    this.setState({readBuckets})
  }

  private handleSelectWriteBucket = (bucketName: string): void => {
    const writeBuckets = selectBucket(bucketName, this.state.writeBuckets)

    this.setState({writeBuckets})
  }

  private handleReadSelectAllBuckets = () => {
    const readBuckets = this.props.buckets.map(b => b.name)
    this.setState({readBuckets})
  }

  private handleReadDeselectAllBuckets = () => {
    this.setState({readBuckets: []})
  }
  j
  private handleWriteSelectAllBuckets = () => {
    const writeBuckets = this.props.buckets.map(b => b.name)
    this.setState({writeBuckets})
  }

  private handleWriteDeselectAllBuckets = () => {
    this.setState({writeBuckets: []})
  }

  private handleSave = async () => {
    const {
      params: {orgID},
      onCreateAuthorization,
    } = this.props
    const {activeTabRead, activeTabWrite} = this.state

    let permissions = []

    if (activeTabRead === BucketTab.Scoped) {
      permissions = [...this.readBucketPermissions]
    } else {
      permissions = [...this.allReadBucketPermissions]
    }

    if (activeTabWrite === BucketTab.Scoped) {
      permissions = [...permissions, ...this.writeBucketPermissions]
    } else {
      permissions = [...permissions, ...this.allWriteBucketPermissions]
    }

    const token: Authorization = {
      orgID,
      description: this.state.description,
      permissions,
    }

    await onCreateAuthorization(token)

    this.handleDismiss()
  }

  private get writeBucketPermissions(): Permission[] {
    const {buckets} = this.props

    const writeBuckets = this.state.writeBuckets.map(bucketName => {
      return buckets.find(b => b.name === bucketName)
    })

    return specificBucketsPermissions(writeBuckets, Permission.ActionEnum.Write)
  }

  private get readBucketPermissions(): Permission[] {
    const {buckets} = this.props

    const readBuckets = this.state.readBuckets.map(bucketName => {
      return buckets.find(b => b.name === bucketName)
    })

    return specificBucketsPermissions(readBuckets, Permission.ActionEnum.Read)
  }

  private get allReadBucketPermissions(): Permission[] {
    const {
      params: {orgID},
    } = this.props

    return allBucketsPermissions(orgID, Permission.ActionEnum.Read)
  }

  private get allWriteBucketPermissions(): Permission[] {
    const {
      params: {orgID},
    } = this.props

    return allBucketsPermissions(orgID, Permission.ActionEnum.Write)
  }

  private handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const {value} = e.target

    this.setState({description: value})
  }

  private handleDismiss = () => {
    const {
      router,
      params: {orgID},
    } = this.props

    router.push(`/orgs/${orgID}/tokens`)
  }
}

const mstp = ({buckets: {list}}: AppState): StateProps => {
  return {buckets: list}
}

const mdtp: DispatchProps = {
  onCreateAuthorization: createAuthorization,
}

export default connect<{}, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter(BucketsTokenOverlay))
