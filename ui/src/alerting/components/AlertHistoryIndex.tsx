// Libraries
import React, {useMemo, useState, FC, createContext} from 'react'
import {Page} from '@influxdata/clockface'
import {connect} from 'react-redux'

// Components
import EventViewer from 'src/eventViewer/components/EventViewer'
import EventTable from 'src/eventViewer/components/EventTable'
import AlertHistoryControls from 'src/alerting/components/AlertHistoryControls'
import AlertHistoryQueryParams from 'src/alerting/components/AlertHistoryQueryParams'

// Constants
import {
  STATUS_FIELDS,
  NOTIFICATION_FIELDS,
} from 'src/alerting/constants/history'

// Utils
import {
  loadStatuses,
  loadNotifications,
  getInitialHistoryType,
  getInitialState,
} from 'src/alerting/utils/history'
import {getResourceIDs} from 'src/alerting/selectors'

// Types
import {AlertHistoryType, AppState} from 'src/types'
import GetResources, {ResourceType} from 'src/shared/components/GetResources'

interface ResourceIDs {
  checkIDs: {[x: string]: boolean}
  endpointIDs: {[x: string]: boolean}
  ruleIDs: {[x: string]: boolean}
}

export const ResourceIDsContext = createContext<ResourceIDs>(null)

interface OwnProps {
  params: {orgID: string}
}

interface StateProps {
  resourceIDs: ResourceIDs
}

type Props = OwnProps & StateProps

const AlertHistoryIndex: FC<Props> = ({params: {orgID}, resourceIDs}) => {
  const [historyType, setHistoryType] = useState<AlertHistoryType>(
    getInitialHistoryType()
  )

  const loadRows = useMemo(() => {
    return historyType === 'statuses'
      ? options => loadStatuses(orgID, options)
      : options => loadNotifications(orgID, options)
  }, [orgID, historyType])

  const fields =
    historyType === 'statuses' ? STATUS_FIELDS : NOTIFICATION_FIELDS

  return (
    <GetResources resource={ResourceType.Checks}>
      <GetResources resource={ResourceType.NotificationEndpoints}>
        <GetResources resource={ResourceType.NotificationRules}>
          <ResourceIDsContext.Provider value={resourceIDs}>
            <EventViewer loadRows={loadRows} initialState={getInitialState()}>
              {props => (
                <Page
                  titleTag="Check Statuses | InfluxDB 2.0"
                  className="alert-history-page"
                >
                  <Page.Header fullWidth={true}>
                    <div className="alert-history-page--header">
                      <Page.Title title="Check Statuses" />
                      <AlertHistoryQueryParams
                        searchInput={props.state.searchInput}
                        historyType={historyType}
                      />
                      <AlertHistoryControls
                        historyType={historyType}
                        onSetHistoryType={setHistoryType}
                        eventViewerProps={props}
                      />
                    </div>
                  </Page.Header>
                  <Page.Contents
                    fullWidth={true}
                    fullHeight={true}
                    scrollable={false}
                    className="alert-history-page--contents"
                  >
                    <div className="alert-history">
                      <EventTable {...props} fields={fields} />
                    </div>
                  </Page.Contents>
                </Page>
              )}
            </EventViewer>
          </ResourceIDsContext.Provider>
        </GetResources>
      </GetResources>
    </GetResources>
  )
}

const mstp = (state: AppState) => {
  const checkIDs = getResourceIDs(state, ResourceType.Checks)
  const endpointIDs = getResourceIDs(state, ResourceType.NotificationEndpoints)
  const ruleIDs = getResourceIDs(state, ResourceType.NotificationRules)

  const resourceIDs = {
    checkIDs,
    endpointIDs,
    ruleIDs,
  }

  return {resourceIDs}
}

export default connect<StateProps>(mstp)(AlertHistoryIndex)
