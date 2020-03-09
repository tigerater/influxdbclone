// Libraries
import React, {FC, useEffect} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'
import {getTimeRangeByDashboardID} from 'src/dashboards/selectors'

// Actions
import * as actions from 'src/dashboards/actions/ranges'

// Types
import {TimeRange, AppState} from 'src/types'

interface StateProps {
  timeRange: TimeRange
}

interface DispatchProps {
  setDashboardTimeRange: typeof actions.setDashboardTimeRange
  updateQueryParams: typeof actions.updateQueryParams
}

type Props = WithRouterProps & StateProps & DispatchProps

const GetTimeRange: FC<Props> = ({
  location,
  params,
  timeRange,
  setDashboardTimeRange,
  updateQueryParams,
}: Props) => {
  const isEditing = location.pathname.includes('edit')
  const isNew = location.pathname.includes('new')

  useEffect(() => {
    if (isEditing || isNew) {
      return
    }

    setDashboardTimeRange(params.dashboardID, timeRange)
    const {lower, upper} = timeRange
    updateQueryParams({
      lower,
      upper,
    })
  }, [isEditing, isNew])

  return <div />
}

const mstp = (state: AppState, props: Props) => {
  const timeRange = getTimeRangeByDashboardID(state, props.params.dashboardID)
  return {timeRange}
}

const mdtp: DispatchProps = {
  updateQueryParams: actions.updateQueryParams,
  setDashboardTimeRange: actions.setDashboardTimeRange,
}

export default withRouter(
  connect<StateProps, DispatchProps>(
    mstp,
    mdtp
  )(GetTimeRange)
)
