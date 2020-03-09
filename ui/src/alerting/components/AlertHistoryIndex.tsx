// Libraries
import React, {useMemo, useState, FC, createContext} from 'react'
import {Page} from '@influxdata/clockface'
import {connect} from 'react-redux'

// Components
import EventViewer from 'src/eventViewer/components/EventViewer'
import EventTable from 'src/eventViewer/components/EventTable'
import AlertHistoryControls from 'src/alerting/components/AlertHistoryControls'
import AlertHistoryQueryParams from 'src/alerting/components/AlertHistoryQueryParams'
import GetResources from 'src/resources/components/GetResources'

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
import {getCheckIDs, getEndpointIDs, getRuleIDs} from 'src/alerting/selectors'

// Types
import {ResourceIDs} from 'src/alerting/reducers/checks'
import {ResourceType, AlertHistoryType, AppState} from 'src/types'

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
    <GetResources
      resources={[
        ResourceType.Checks,
        ResourceType.NotificationEndpoints,
        ResourceType.NotificationRules,
      ]}
    >
      <ResourceIDsContext.Provider value={resourceIDs}>
        <EventViewer loadRows={loadRows} initialState={getInitialState()}>
          {props => (
            <Page
              titleTag="Check Statuses | InfluxDB 2.0"
              className="alert-history-page"
            >
              <Page.Header fullWidth={true}>
                <div className="alert-history-page--header">
                  <Page.Title
                    title="Check Statuses"
                    testID="alert-history-title"
                  />
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
  )
}

const mstp = (state: AppState) => {
  const checkIDs = getCheckIDs(state)
  const endpointIDs = getEndpointIDs(state)
  const ruleIDs = getRuleIDs(state)

  const resourceIDs = {
    checkIDs,
    endpointIDs,
    ruleIDs,
  }

  return {resourceIDs}
}

export default connect<StateProps>(mstp)(AlertHistoryIndex)
