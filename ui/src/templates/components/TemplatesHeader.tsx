// Libraries
import React, {PureComponent} from 'react'

// Components
import {Page} from 'src/pageLayout'
import PageTitleWithOrg from 'src/shared/components/PageTitleWithOrg'

// Components
import {Tabs} from 'src/clockface'
import {Button, IconFont, ComponentColor} from '@influxdata/clockface'

interface Props {
  onImportTemplate: () => void
  showOrgDropdown?: boolean
  isFullPage?: boolean
  filterComponent: () => JSX.Element
}

export default class TemplatesHeader extends PureComponent<Props> {
  public static defaultProps: {
    showOrgDropdown: boolean
    isFullPage: boolean
  } = {
    showOrgDropdown: true,
    isFullPage: true,
  }

  public render() {
    const {isFullPage, filterComponent} = this.props

    if (isFullPage) {
      return (
        <Page.Header fullWidth={false}>
          <Page.Header.Left>
            <PageTitleWithOrg title="Templates" />
          </Page.Header.Left>
          <Page.Header.Right>{this.importButton}</Page.Header.Right>
        </Page.Header>
      )
    }

    return (
      <Tabs.TabContentsHeader>
        {filterComponent()}
        {this.importButton}
      </Tabs.TabContentsHeader>
    )
  }

  private get importButton(): JSX.Element {
    return (
      <Button
        text="Import Template"
        icon={IconFont.Plus}
        color={ComponentColor.Primary}
        onClick={this.props.onImportTemplate}
      />
    )
  }
}
