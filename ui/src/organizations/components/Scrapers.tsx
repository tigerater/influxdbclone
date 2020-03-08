// Libraries
import _ from 'lodash'
import React, {PureComponent, ChangeEvent} from 'react'

// APIs
import {client} from 'src/utils/api'

// Components
import ScraperList from 'src/organizations/components/ScraperList'
import {
  Button,
  ComponentColor,
  IconFont,
  ComponentSize,
  EmptyState,
  Input,
  InputType,
  Tabs,
} from 'src/clockface'
import DataLoadersWizard from 'src/dataLoaders/components/DataLoadersWizard'

// Decorators
import {ErrorHandling} from 'src/shared/decorators/errors'

// Types
import {ScraperTargetResponse, Bucket} from '@influxdata/influx'
import {OverlayState} from 'src/types/v2'
import {DataLoaderType, DataLoaderStep} from 'src/types/v2/dataLoaders'

interface Props {
  scrapers: ScraperTargetResponse[]
  onChange: () => void
  orgName: string
  buckets: Bucket[]
}

interface State {
  overlayState: OverlayState
  searchTerm: string
}

@ErrorHandling
export default class Scrapers extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)

    this.state = {
      overlayState: OverlayState.Closed,
      searchTerm: '',
    }
  }

  public render() {
    const {searchTerm} = this.state

    return (
      <>
        <Tabs.TabContentsHeader>
          <Input
            icon={IconFont.Search}
            placeholder="Filter scrapers by bucket..."
            widthPixels={290}
            value={searchTerm}
            type={InputType.Text}
            onChange={this.handleFilterChange}
            onBlur={this.handleFilterBlur}
          />
          {this.createScraperButton}
        </Tabs.TabContentsHeader>
        <ScraperList
          scrapers={this.configurations}
          emptyState={this.emptyState}
          onDeleteScraper={this.handleDeleteScraper}
          onUpdateScraper={this.handleUpdateScraper}
        />
        <DataLoadersWizard
          visible={this.isOverlayVisible}
          onCompleteSetup={this.handleDismissDataLoaders}
          startingType={DataLoaderType.Scraping}
          startingStep={DataLoaderStep.Configure}
          buckets={this.buckets}
        />
      </>
    )
  }

  private get buckets(): Bucket[] {
    const {buckets} = this.props

    if (!buckets || !buckets.length) {
      return []
    }
    return buckets
  }

  private get configurations(): ScraperTargetResponse[] {
    const {scrapers} = this.props
    const {searchTerm} = this.state

    if (!scrapers || !scrapers) {
      return []
    }

    return scrapers.filter(c => {
      if (!searchTerm) {
        return true
      }
      if (!c.bucket) {
        return false
      }

      return String(c.bucket)
        .toLocaleLowerCase()
        .includes(searchTerm.toLocaleLowerCase())
    })
  }

  private get isOverlayVisible(): boolean {
    return this.state.overlayState === OverlayState.Open
  }

  private get createScraperButton(): JSX.Element {
    return (
      <Button
        text="Create Scraper"
        icon={IconFont.Plus}
        color={ComponentColor.Primary}
        onClick={this.handleAddScraper}
      />
    )
  }

  private handleAddScraper = () => {
    this.setState({overlayState: OverlayState.Open})
  }

  private handleDismissDataLoaders = () => {
    this.setState({overlayState: OverlayState.Closed})
    this.props.onChange()
  }

  private get emptyState(): JSX.Element {
    const {orgName} = this.props
    const {searchTerm} = this.state

    if (_.isEmpty(searchTerm)) {
      return (
        <EmptyState size={ComponentSize.Medium}>
          <EmptyState.Text
            text={`${orgName} does not own any Scrapers, why not create one?`}
            highlightWords={['Scrapers']}
          />
          {this.createScraperButton}
        </EmptyState>
      )
    }

    return (
      <EmptyState size={ComponentSize.Medium}>
        <EmptyState.Text text="No Scraper buckets match your query" />
      </EmptyState>
    )
  }

  private handleUpdateScraper = async (scraper: ScraperTargetResponse) => {
    await client.scrapers.update(scraper.id, scraper)
    this.props.onChange()
  }

  private handleDeleteScraper = async (scraper: ScraperTargetResponse) => {
    await client.scrapers.delete(scraper.id)
    this.props.onChange()
  }

  private handleFilterChange = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }

  private handleFilterBlur = (e: ChangeEvent<HTMLInputElement>): void => {
    this.setState({searchTerm: e.target.value})
  }
}
