// Libraries
import React, {PureComponent} from 'react'
import {Page} from 'src/pageLayout'

// Components
import {
  SlideToggle,
  ComponentSize,
  ComponentSpacer,
  FlexDirection,
  JustifyContent,
} from '@influxdata/clockface'
import {Tabs, ComponentStatus} from 'src/clockface'
import AddResourceDropdown from 'src/shared/components/AddResourceDropdown'
import PageTitleWithOrg from 'src/shared/components/PageTitleWithOrg'

// Types
import {LimitStatus} from 'src/cloud/actions/limits'

interface Props {
  onCreateTask: () => void
  setShowInactive: () => void
  showInactive: boolean
  onImportTask: () => void
  showOrgDropdown?: boolean
  isFullPage?: boolean
  filterComponent: () => JSX.Element
  limitStatus: LimitStatus
  onImportFromTemplate: () => void
}

export default class TasksHeader extends PureComponent<Props> {
  public static defaultProps: {
    showOrgDropdown: boolean
    isFullPage: boolean
  } = {
    showOrgDropdown: true,
    isFullPage: true,
  }

  public render() {
    const {
      onCreateTask,
      setShowInactive,
      showInactive,
      onImportTask,
      isFullPage,
      filterComponent,
      onImportFromTemplate,
    } = this.props

    if (isFullPage) {
      return (
        <Page.Header fullWidth={false}>
          <Page.Header.Left>
            <PageTitleWithOrg title={this.pageTitle} />
          </Page.Header.Left>
          <Page.Header.Right>
            <SlideToggle.Label text="Show Inactive" />
            <SlideToggle
              active={showInactive}
              size={ComponentSize.ExtraSmall}
              onChange={setShowInactive}
            />
            <AddResourceDropdown
              canImportFromTemplate={true}
              onSelectNew={onCreateTask}
              onSelectImport={onImportTask}
              onSelectTemplate={onImportFromTemplate}
              resourceName="Task"
              status={this.addResourceStatus}
            />
          </Page.Header.Right>
        </Page.Header>
      )
    }

    return (
      <Tabs.TabContentsHeader>
        {filterComponent()}
        <ComponentSpacer
          margin={ComponentSize.Small}
          direction={FlexDirection.Row}
          justifyContent={JustifyContent.FlexEnd}
        >
          <SlideToggle.Label text="Show Inactive" />
          <SlideToggle
            active={showInactive}
            size={ComponentSize.ExtraSmall}
            onChange={setShowInactive}
            testID="tasks-header--toggle-active"
          />
          <AddResourceDropdown
            onSelectNew={onCreateTask}
            onSelectImport={onImportTask}
            resourceName="Task"
          />
        </ComponentSpacer>
      </Tabs.TabContentsHeader>
    )
  }

  private get pageTitle() {
    const {showOrgDropdown} = this.props

    if (showOrgDropdown) {
      return 'Tasks'
    }
    return ''
  }

  private get addResourceStatus(): ComponentStatus {
    const {limitStatus} = this.props
    if (limitStatus === LimitStatus.EXCEEDED) {
      return ComponentStatus.Disabled
    }
    return ComponentStatus.Default
  }
}
