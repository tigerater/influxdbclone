// Libraries
import React, {FunctionComponent, useEffect} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {Overlay, SpinnerContainer, TechnoSpinner} from '@influxdata/clockface'
import CheckEOHeader from 'src/alerting/components/CheckEOHeader'
import TimeMachine from 'src/timeMachine/components/TimeMachine'

// Actions
import {saveCheckFromTimeMachine} from 'src/alerting/actions/checks'
import {
  setActiveTimeMachine,
  updateTimeMachineCheck,
  setTimeMachineCheck,
} from 'src/timeMachine/actions'
import {createCheckFailed} from 'src/shared/copy/notifications'
import {notify} from 'src/shared/actions/notifications'

// Utils
import {createView} from 'src/shared/utils/view'
import {getActiveTimeMachine} from 'src/timeMachine/selectors'

// Types
import {Check, AppState, RemoteDataState, CheckViewProperties} from 'src/types'
import {DEFAULT_DEADMAN_CHECK} from 'src/alerting/constants'

interface DispatchProps {
  setTimeMachineCheck: typeof setTimeMachineCheck
  updateTimeMachineCheck: typeof updateTimeMachineCheck
  onSetActiveTimeMachine: typeof setActiveTimeMachine
  saveCheckFromTimeMachine: typeof saveCheckFromTimeMachine
  notify: typeof notify
}

interface StateProps {
  check: Partial<Check>
  checkStatus: RemoteDataState
}

type Props = DispatchProps & StateProps & WithRouterProps

const NewCheckOverlay: FunctionComponent<Props> = ({
  onSetActiveTimeMachine,
  updateTimeMachineCheck,
  setTimeMachineCheck,
  saveCheckFromTimeMachine,
  params,
  router,
  checkStatus,
  check,
  notify,
}) => {
  useEffect(() => {
    const view = createView<CheckViewProperties>('deadman')
    onSetActiveTimeMachine('alerting', {
      view,
      alerting: {
        checkStatus: RemoteDataState.Done,
        check: DEFAULT_DEADMAN_CHECK,
      },
    })
  }, [])

  const handleUpdateName = (name: string) => {
    updateTimeMachineCheck({name})
  }

  const handleClose = () => {
    setTimeMachineCheck(RemoteDataState.NotStarted, null)
    router.push(`/orgs/${params.orgID}/alerting`)
  }

  const handleSave = async () => {
    // todo: when check has own view
    // save view as view
    // put view.id on check.viewID
    try {
      await saveCheckFromTimeMachine()
      handleClose()
    } catch (e) {
      console.error(e)
      notify(createCheckFailed(e.message))
    }
  }

  return (
    <Overlay visible={true} className="veo-overlay">
      <div className="veo">
        <SpinnerContainer
          spinnerComponent={<TechnoSpinner />}
          loading={checkStatus || RemoteDataState.Loading}
        >
          <CheckEOHeader
            key={check && check.name}
            name={check && check.name}
            onSetName={handleUpdateName}
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
  const {
    alerting: {check, checkStatus},
  } = getActiveTimeMachine(state)

  return {check, checkStatus}
}

const mdtp: DispatchProps = {
  setTimeMachineCheck: setTimeMachineCheck,
  updateTimeMachineCheck: updateTimeMachineCheck,
  onSetActiveTimeMachine: setActiveTimeMachine,
  saveCheckFromTimeMachine: saveCheckFromTimeMachine,
  notify: notify,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter(NewCheckOverlay))
