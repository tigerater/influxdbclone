// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'
import _ from 'lodash'

// Components
import {Input, Button, EmptyState} from '@influxdata/clockface'
import {Tabs, Sort} from 'src/clockface'
import ScraperList from 'src/scrapers/components/ScraperList'
import NoBucketsWarning from 'src/buckets/components/NoBucketsWarning'

// Actions
import {updateScraper, deleteScraper} from 'src/scrapers/actions'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'
import {SortTypes} from 'src/shared/utils/sort'

// Types
import {ScraperTargetResponse} from '@influxdata/influx'
import {
  IconFont,
  InputType,
  ComponentSize,
  ComponentColor,
  ComponentStatus,
} from '@influxdata/clockface'
import {AppState, Bucket} from 'src/types'
import FilterList from 'src/shared/components/Filter'

interface StateProps {
  scrapers: ScraperTargetResponse[]
  buckets: Bucket[]
}

interface DispatchProps {
  onUpdateScraper: typeof updateScraper
  onDeleteScraper: typeof deleteScraper
}

interface OwnProps {
  orgName: string
}

type Props = OwnProps & StateProps & DispatchProps & WithRouterProps

interface State {
  searchTerm: string
  sortKey: SortKey
  sortDirection: Sort
  sortType: SortTypes
}

type SortKey = keyof ScraperTargetResponse

@ErrorHandling
class Scrapers extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)

    this.state = {
      searchTerm: '',
      sortKey: 'name',
      sortDirection: Sort.Ascending,
      sortType: SortTypes.String,
    }
  }

  public render() {
    const {searchTerm, sortKey, sortDirection, sortType} = this.state
    const {scrapers} = this.props

    return (
      <>
        <Tabs.TabContentsHeader>
          <Input
            icon={IconFont.Search}
            placeholder="Filter scrapers..."
            widthPixels={290}
            value={searchTerm}
            type={InputType.Text}
            onChange={this.handleFilterChange}
            onBlur={this.handleFilterBlur}
          />
          {this.createScraperButton('create-scraper-button-header')}
        </Tabs.TabContentsHeader>
        <NoBucketsWarning visible={this.hasNoBuckets} resourceName="Scrapers" />
        <FilterList<ScraperTargetResponse>
          searchTerm={searchTerm}
          searchKeys={['name', 'url']}
          list={scrapers}
        >
          {sl => (
            <ScraperList
              scrapers={sl}
              emptyState={this.emptyState}
              onDeleteScraper={this.handleDeleteScraper}
              onUpdateScraper={this.handleUpdateScraper}
              sortKey={sortKey}
              sortDirection={sortDirection}
              sortType={sortType}
              onClickColumn={this.handleClickColumn}
            />
          )}
        </FilterList>
      </>
    )
  }

  private handleClickColumn = (nextSort: Sort, sortKey: SortKey) => {
    const sortType = SortTypes.String
    this.setState({sortKey, sortDirection: nextSort, sortType})
  }

  private get hasNoBuckets(): boolean {
    const {buckets} = this.props

    if (!buckets || !buckets.length) {
      return true
    }

    return false
  }

  private createScraperButton = (testID: string): JSX.Element => {
    let status = ComponentStatus.Default
    let titleText = 'Create a new Scraper'

    if (this.hasNoBuckets) {
      status = ComponentStatus.Disabled
      titleText = 'You need at least 1 bucket in order to create a scraper'
    }

    return (
      <Button
        text="Create Scraper"
        icon={IconFont.Plus}
        color={ComponentColor.Primary}
        onClick={this.handleShowOverlay}
        status={status}
        titleText={titleText}
        testID={testID}
      />
    )
  }

  private get emptyState(): JSX.Element {
    const {orgName} = this.props
    const {searchTerm} = this.state

    if (_.isEmpty(searchTerm)) {
      return (
        <EmptyState size={ComponentSize.Large}>
          <EmptyState.Text
            text={`${orgName} does not own any Scrapers , why not create one?`}
            highlightWords={['Scrapers']}
          />
          {this.createScraperButton('create-scraper-button-empty')}
        </EmptyState>
      )
    }

    return (
      <EmptyState size={ComponentSize.Large}>
        <EmptyState.Text text="No Scrapers match your query" />
      </EmptyState>
    )
  }

  private handleUpdateScraper = async (scraper: ScraperTargetResponse) => {
    const {onUpdateScraper} = this.props
    onUpdateScraper(scraper)
  }

  private handleDeleteScraper = async (scraper: ScraperTargetResponse) => {
    const {onDeleteScraper} = this.props
    onDeleteScraper(scraper)
  }

  private handleShowOverlay = (): void => {
    const {
      router,
      params: {orgID},
    } = this.props

    if (this.hasNoBuckets) {
      return
    }

    router.push(`/orgs/${orgID}/scrapers/new`)
  }

  private handleFilterChange = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }

  private handleFilterBlur = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }
}

const mstp = ({scrapers, buckets}: AppState): StateProps => ({
  scrapers: scrapers.list,
  buckets: buckets.list,
})

const mdtp: DispatchProps = {
  onDeleteScraper: deleteScraper,
  onUpdateScraper: updateScraper,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<OwnProps>(Scrapers))
