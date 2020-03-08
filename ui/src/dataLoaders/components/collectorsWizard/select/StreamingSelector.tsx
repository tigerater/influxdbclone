// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import uuid from 'uuid'
import _ from 'lodash'

// Components
import {Input, EmptyState, FormElement, Grid} from '@influxdata/clockface'
import {ResponsiveGridSizer} from 'src/clockface'
import {ErrorHandling} from 'src/shared/decorators/errors'
import CardSelectCard from 'src/clockface/components/card_select/CardSelectCard'

// Constants
import {
  PLUGIN_BUNDLE_OPTIONS,
  BUNDLE_LOGOS,
} from 'src/dataLoaders/constants/pluginConfigs'
import BucketDropdown from 'src/dataLoaders/components/BucketsDropdown'

// Types
import {TelegrafPlugin, BundleName} from 'src/types/dataLoaders'
import {Bucket} from '@influxdata/influx'
import {IconFont, Columns, ComponentSize} from '@influxdata/clockface'

export interface Props {
  buckets: Bucket[]
  selectedBucketName: string
  pluginBundles: BundleName[]
  telegrafPlugins: TelegrafPlugin[]
  onTogglePluginBundle: (telegrafPlugin: string, isSelected: boolean) => void
  onSelectBucket: (bucket: Bucket) => void
}

interface State {
  gridSizerUpdateFlag: string
  searchTerm: string
}

@ErrorHandling
class StreamingSelector extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = {
      gridSizerUpdateFlag: uuid.v4(),
      searchTerm: '',
    }
  }

  public componentDidUpdate(prevProps) {
    const addFirst =
      prevProps.telegrafPlugins.length === 0 &&
      this.props.telegrafPlugins.length > 0

    const removeLast =
      prevProps.telegrafPlugins.length > 0 &&
      this.props.telegrafPlugins.length === 0

    if (addFirst || removeLast) {
      const gridSizerUpdateFlag = uuid.v4()
      this.setState({gridSizerUpdateFlag})
    }
  }

  public render() {
    const {buckets} = this.props
    const {searchTerm} = this.state

    return (
      <div className="wizard-step--grid-container">
        <Grid.Row>
          <Grid.Column widthSM={Columns.Five}>
            <FormElement label="Bucket">
              <BucketDropdown
                selectedBucketID={this.selectedBucketID}
                buckets={buckets}
                onSelectBucket={this.handleSelectBucket}
              />
            </FormElement>
          </Grid.Column>
          <Grid.Column widthSM={Columns.Five} offsetSM={Columns.Two}>
            <FormElement label="">
              <Input
                className="wizard-step--filter"
                size={ComponentSize.Small}
                icon={IconFont.Search}
                value={searchTerm}
                onBlur={this.handleFilterBlur}
                onChange={this.handleFilterChange}
                placeholder="Filter Plugins..."
              />
            </FormElement>
          </Grid.Column>
        </Grid.Row>
        <ResponsiveGridSizer columns={5}>
          {this.filteredBundles.map(b => {
            return (
              <CardSelectCard
                key={b}
                id={b}
                name={b}
                label={b}
                checked={this.isCardChecked(b)}
                onClick={this.handleToggle(b)}
                image={BUNDLE_LOGOS[b]}
              />
            )
          })}
        </ResponsiveGridSizer>
        {this.emptyState}
      </div>
    )
  }

  private get selectedBucketID(): string {
    const {buckets, selectedBucketName} = this.props

    return buckets.find(b => b.name === selectedBucketName).id
  }

  private handleSelectBucket = (bucket: Bucket) => {
    this.props.onSelectBucket(bucket)
  }

  private get emptyState(): JSX.Element {
    const {searchTerm} = this.state

    const noMatches = this.filteredBundles.length === 0

    if (searchTerm && noMatches) {
      return (
        <EmptyState size={ComponentSize.Medium}>
          <EmptyState.Text text="No plugins match your search" />
        </EmptyState>
      )
    }
  }

  private get filteredBundles(): BundleName[] {
    const {searchTerm} = this.state

    return PLUGIN_BUNDLE_OPTIONS.filter(b =>
      b.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }

  private isCardChecked(bundle: BundleName): boolean {
    const {pluginBundles} = this.props

    if (pluginBundles.find(b => b === bundle)) {
      return true
    }
    return false
  }

  private handleToggle = (bundle: BundleName) => (): void => {
    this.props.onTogglePluginBundle(bundle, this.isCardChecked(bundle))
  }

  private handleFilterChange = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }

  private handleFilterBlur = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }
}

export default StreamingSelector
