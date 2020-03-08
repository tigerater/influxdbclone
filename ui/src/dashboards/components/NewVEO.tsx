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
import {saveCurrentCheck} from 'src/alerting/actions/checks'
import {setActiveTimeMachine} from 'src/timeMachine/actions'
import {setName} from 'src/timeMachine/actions'
import {saveVEOView} from 'src/dashboards/actions'

// Utils
import {getActiveTimeMachine} from 'src/timeMachine/selectors'
import {createView} from 'src/shared/utils/view'

// Types
import {AppState, XYViewProperties, RemoteDataState, View} from 'src/types'
import {TimeMachineID} from 'src/timeMachine/constants'

interface DispatchProps {
  onSetActiveTimeMachine: typeof setActiveTimeMachine
  saveCurrentCheck: typeof saveCurrentCheck
  onSetName: typeof setName
  onSaveView: typeof saveVEOView
}

interface StateProps {
  activeTimeMachineID: TimeMachineID
  view: View
}

type Props = DispatchProps & StateProps & WithRouterProps

const NewViewVEO: FunctionComponent<Props> = ({
  onSetActiveTimeMachine,
  activeTimeMachineID,
  saveCurrentCheck,
  onSaveView,
  onSetName,
  params: {orgID, dashboardID},
  router,
  view,
}) => {
  useEffect(() => {
    const view = createView<XYViewProperties>('xy')
    onSetActiveTimeMachine('veo', {view})
  }, [])

  const handleClose = () => {
    router.push(`/orgs/${orgID}/dashboards/${dashboardID}`)
  }

  const handleSave = () => {
    try {
      if (view.properties.type === 'check') {
        saveCurrentCheck()
      }
      onSaveView(dashboardID)
      handleClose()
    } catch (e) {}
  }

  let loadingState = RemoteDataState.Loading
  const viewIsNew = !get(view, 'id', null)
  if (activeTimeMachineID === 'veo' && viewIsNew) {
    loadingState = RemoteDataState.Done
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

const mstp = (state: AppState): StateProps => {
  const {activeTimeMachineID} = state.timeMachines
  const {view} = getActiveTimeMachine(state)

  return {view, activeTimeMachineID}
}

const mdtp: DispatchProps = {
  onSetName: setName,
  onSaveView: saveVEOView,
  saveCurrentCheck: saveCurrentCheck,
  onSetActiveTimeMachine: setActiveTimeMachine,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter(NewViewVEO))
