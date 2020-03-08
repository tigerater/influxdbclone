// Libraries
import React, {FunctionComponent, useEffect} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'
import {get} from 'lodash'

// Components
import {Overlay, SpinnerContainer, TechnoSpinner} from '@influxdata/clockface'
import TimeMachine from 'src/timeMachine/components/TimeMachine'
import VEOHeader from 'src/dashboards/components/VEOHeader'

// Actions
import {setActiveTimeMachine} from 'src/timeMachine/actions'
import {setName} from 'src/timeMachine/actions'
import {saveVEOView} from 'src/dashboards/actions'
import {setView, getViewForTimeMachine} from 'src/dashboards/actions/views'

// Utils
import {TimeMachineEnum} from 'src/timeMachine/constants'
import {getView} from 'src/dashboards/selectors'

// Types
import {AppState, RemoteDataState, QueryView} from 'src/types'
import {executeQueries} from 'src/timeMachine/actions/queries'

interface DispatchProps {
  onSetActiveTimeMachine: typeof setActiveTimeMachine
  onSetName: typeof setName
  onSaveView: typeof saveVEOView
  setView: typeof setView
  executeQueries: typeof executeQueries
  getViewForTimeMachine: typeof getViewForTimeMachine
}

interface StateProps {
  view: QueryView | null
  loadingState: RemoteDataState
}

type Props = DispatchProps & StateProps & WithRouterProps

const EditViewVEO: FunctionComponent<Props> = ({
  onSetActiveTimeMachine,
  getViewForTimeMachine,
  executeQueries,
  loadingState,
  onSaveView,
  onSetName,
  params: {orgID, cellID, dashboardID},
  router,
  view,
}) => {
  useEffect(() => {
    if (view) {
      onSetActiveTimeMachine(TimeMachineEnum.VEO, {view})
    } else {
      getViewForTimeMachine(dashboardID, cellID, TimeMachineEnum.VEO)
    }
  }, [view, orgID, cellID, dashboardID])

  useEffect(() => {
    executeQueries()
  }, [view])

  const handleClose = () => {
    router.push(`/orgs/${orgID}/dashboards/${dashboardID}`)
  }

  const handleSave = () => {
    try {
      onSaveView(dashboardID)
      handleClose()
    } catch (e) {}
  }

  return (
    <Overlay visible={true} className="veo-overlay">
      <div className="veo">
        <SpinnerContainer
          spinnerComponent={<TechnoSpinner />}
          loading={loadingState}
        >
          <VEOHeader
            key={view && view.name}
            name={view && view.name}
            onSetName={onSetName}
            onCancel={handleClose}
            onSave={handleSave}
          />
          <div className="veo-contents">
            <TimeMachine />
          </div>
        </SpinnerContainer>
      </div>
    </Overlay>
  )
}

const mstp = (state: AppState, {params: {cellID}}): StateProps => {
  const {activeTimeMachineID} = state.timeMachines

  const view = getView(state, cellID) as QueryView

  const viewMatchesRoute = get(view, 'id', null) === cellID

  let loadingState = RemoteDataState.Loading

  if (activeTimeMachineID === TimeMachineEnum.VEO && viewMatchesRoute) {
    loadingState = RemoteDataState.Done
  }

  return {view, loadingState}
}

const mdtp: DispatchProps = {
  onSetName: setName,
  setView: setView,
  onSaveView: saveVEOView,
  onSetActiveTimeMachine: setActiveTimeMachine,
  executeQueries: executeQueries,
  getViewForTimeMachine: getViewForTimeMachine,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter<StateProps & DispatchProps>(EditViewVEO))
