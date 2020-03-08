// Libraries
import React, {Component} from 'react'
import {connect} from 'react-redux'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import SettingsNavigation from 'src/settings/components/SettingsNavigation'
import SettingsHeader from 'src/settings/components/SettingsHeader'
import {Tabs} from 'src/clockface'
import {Page} from 'src/pageLayout'
import TabbedPageSection from 'src/shared/components/tabbed_page/TabbedPageSection'
import VariablesTab from 'src/variables/components/VariablesTab'
import GetResources, {ResourceTypes} from 'src/shared/components/GetResources'

// Types
import {AppState, Organization} from 'src/types'

interface StateProps {
  org: Organization
}

@ErrorHandling
class VariablesIndex extends Component<StateProps> {
  public render() {
    const {org, children} = this.props

    return (
      <>
        <Page titleTag={org.name}>
          <SettingsHeader />
          <Page.Contents fullWidth={false} scrollable={true}>
            <div className="col-xs-12">
              <Tabs>
                <SettingsNavigation tab="variables" orgID={org.id} />
                <Tabs.TabContents>
                  <TabbedPageSection
                    id="settings-tab--variables"
                    url="variables"
                    title="Variables"
                  >
                    <GetResources resource={ResourceTypes.Variables}>
                      <VariablesTab />
                    </GetResources>
                  </TabbedPageSection>
                </Tabs.TabContents>
              </Tabs>
            </div>
          </Page.Contents>
        </Page>
        {children}
      </>
    )
  }
}

const mstp = ({orgs: {org}}: AppState): StateProps => ({org})

export default connect<StateProps, {}, {}>(
  mstp,
  null
)(VariablesIndex)
