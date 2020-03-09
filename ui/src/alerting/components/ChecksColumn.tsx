// Libraries
import React, {FunctionComponent} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'

// Selectors
import {viewableLabels} from 'src/labels/selectors'

// Components
import CheckCards from 'src/alerting/components/CheckCards'
import AlertsColumn from 'src/alerting/components/AlertsColumn'
import CreateCheckDropdown from 'src/alerting/components/CreateCheckDropdown'

// Types
import {Check, NotificationRuleDraft, AppState} from 'src/types'

interface StateProps {
  checks: Check[]
  rules: NotificationRuleDraft[]
  endpoints: AppState['endpoints']['list']
}

type Props = StateProps & WithRouterProps

const ChecksColumn: FunctionComponent<Props> = ({
  checks,
  router,
  params: {orgID},
  rules,
  endpoints,
}) => {
  const handleCreateThreshold = () => {
    router.push(`/orgs/${orgID}/alerting/checks/new-threshold`)
  }

  const handleCreateDeadman = () => {
    router.push(`/orgs/${orgID}/alerting/checks/new-deadman`)
  }

  const tooltipContents = (
    <>
      A <strong>Check</strong> is a periodic query that the system
      <br />
      performs against your time series data
      <br />
      that will generate a status
      <br />
      <br />
      <a
        href="https://v2.docs.influxdata.com/v2.0/monitor-alert/checks/create/"
        target="_blank"
      >
        Read Documentation
      </a>
    </>
  )

  const noAlertingResourcesExist =
    !checks.length && !rules.length && !endpoints.length

  const createButton = (
    <CreateCheckDropdown
      onCreateThreshold={handleCreateThreshold}
      onCreateDeadman={handleCreateDeadman}
    />
  )

  return (
    <AlertsColumn
      title="Checks"
      createButton={createButton}
      questionMarkTooltipContents={tooltipContents}
    >
      {searchTerm => (
        <CheckCards
          checks={checks}
          searchTerm={searchTerm}
          onCreateThreshold={handleCreateThreshold}
          onCreateDeadman={handleCreateDeadman}
          showFirstTimeWidget={noAlertingResourcesExist}
        />
      )}
    </AlertsColumn>
  )
}

const mstp = (state: AppState) => {
  const {
    checks: {list: checks},
    labels: {list: labels},
    rules: {list: rules},
    endpoints,
  } = state

  return {
    checks,
    labels: viewableLabels(labels),
    rules,
    endpoints: endpoints.list,
  }
}

export default connect<StateProps, {}, {}>(
  mstp,
  null
)(withRouter(ChecksColumn))
