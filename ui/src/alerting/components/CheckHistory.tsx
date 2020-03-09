// Libraries
import React, {useMemo, FC} from 'react'
import {connect} from 'react-redux'

// Components
import {Page} from '@influxdata/clockface'
import EventViewer from 'src/eventViewer/components/EventViewer'
import CheckHistoryControls from 'src/alerting/components/CheckHistoryControls'
import CheckHistoryVisualization from 'src/alerting/components/CheckHistoryVisualization'
import AlertHistoryQueryParams from 'src/alerting/components/AlertHistoryQueryParams'
import EventTable from 'src/eventViewer/components/EventTable'

//Context
import {ResourceIDsContext} from 'src/alerting/components/AlertHistoryIndex'

// Constants
import {STATUS_FIELDS} from 'src/alerting/constants/history'

// Utils
import {loadStatuses, getInitialState} from 'src/alerting/utils/history'
import {getCheckIDs} from 'src/alerting/selectors'

// Types
import {ResourceIDs} from 'src/alerting/reducers/checks'
import GetResources, {ResourceType} from 'src/shared/components/GetResources'
import {AppState, Check, TimeZone} from 'src/types'

interface OwnProps {
  params: {orgID: string; checkID: string}
}

interface StateProps {
  check: Check
  timeZone: TimeZone
  resourceIDs: ResourceIDs
}

type Props = OwnProps & StateProps

const CheckHistory: FC<Props> = ({
  params: {orgID},
  check,
  timeZone,
  resourceIDs,
}) => {
  const loadRows = useMemo(() => options => loadStatuses(orgID, options), [
    orgID,
  ])
  const historyType = 'statuses'
  const fields = STATUS_FIELDS
  return (
    <GetResources resources={[ResourceType.Checks]}>
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
                  <CheckHistoryControls eventViewerProps={props} />
                </div>
              </Page.Header>
              <Page.Contents
                fullWidth={true}
                scrollable={false}
                className="alert-history-page--contents"
              >
                <div className="alert-history-contents">
                  {check.type !== 'custom' && (
                    <CheckHistoryVisualization
                      check={check}
                      timeZone={timeZone}
                    />
                  )}
                  <div className="alert-history">
                    <EventTable {...props} fields={fields} />
                  </div>
                </div>
              </Page.Contents>
            </Page>
          )}
        </EventViewer>
      </ResourceIDsContext.Provider>
    </GetResources>
  )
}

const mstp = (state: AppState, props: OwnProps) => {
  const check = state.checks.list.find(({id}) => id === props.params.checkID)
  const timeZone = state.app.persisted.timeZone
  const checkIDs = getCheckIDs(state)

  const resourceIDs = {
    checkIDs,
    endpointIDs: null,
    ruleIDs: null,
  }

  return {check, timeZone, resourceIDs}
}

export default connect<StateProps>(mstp)(CheckHistory)
