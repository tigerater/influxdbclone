import {produce} from 'immer'

//Types
import {Actions, ActionTypes} from 'src/cloud/actions/limits'
import {RemoteDataState} from 'src/types'
import {LimitStatus} from 'src/cloud/actions/limits'

interface Limit {
  maxAllowed: number
  limitStatus: LimitStatus
}

export interface LimitsState {
  dashboards: Limit
  tasks: Limit
  buckets: Limit
  status: RemoteDataState
}

const defaultLimit: Limit = {
  maxAllowed: Infinity,
  limitStatus: LimitStatus.OK,
}

export const defaultState: LimitsState = {
  dashboards: defaultLimit,
  tasks: defaultLimit,
  buckets: defaultLimit,
  status: RemoteDataState.NotStarted,
}

export const limitsReducer = (
  state = defaultState,
  action: Actions
): LimitsState =>
  produce(state, draftState => {
    switch (action.type) {
      case ActionTypes.SetLimitsStatus: {
        const {status} = action.payload
        draftState.status = status
        return
      }

      case ActionTypes.SetLimits: {
        const {limits} = action.payload

        const {maxBuckets} = limits.bucket
        const {maxDashboards} = limits.dashboard
        const {maxTasks} = limits.task

        draftState.buckets.maxAllowed = maxBuckets
        draftState.dashboards.maxAllowed = maxDashboards
        draftState.tasks.maxAllowed = maxTasks
        return
      }
      case ActionTypes.SetDashboardLimitStatus: {
        draftState.dashboards.limitStatus = action.payload.limitStatus
        return
      }
      case ActionTypes.SetBucketLimitStatus: {
        draftState.buckets.limitStatus = action.payload.limitStatus
        return
      }
      case ActionTypes.SetTaskLimitStatus: {
        draftState.tasks.limitStatus = action.payload.limitStatus
        return
      }
    }
  })

export default limitsReducer
