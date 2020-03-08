// Libraries
import {Dispatch} from 'react'

// Constants
import * as copy from 'src/shared/copy/notifications'

// APIs
import * as api from 'src/client'

// Utils
import {getActiveTimeMachine} from 'src/timeMachine/selectors'

//Actions
import {
  notify,
  Action as NotificationAction,
} from 'src/shared/actions/notifications'

// Types
import {RemoteDataState} from '@influxdata/clockface'
import {Check, GetState, CheckType} from 'src/types'

export type Action =
  | ReturnType<typeof setAllChecks>
  | ReturnType<typeof setCheck>
  | ReturnType<typeof removeCheck>
  | ReturnType<typeof setCurrentCheck>
  | ReturnType<typeof setCurrentCheckStatus>
  | ReturnType<typeof updateCurrentCheck>
  | ReturnType<typeof changeCurrentCheckType>

export const setAllChecks = (status: RemoteDataState, checks?: Check[]) => ({
  type: 'SET_ALL_CHECKS' as 'SET_ALL_CHECKS',
  payload: {status, checks},
})

export const setCheck = (check: Check) => ({
  type: 'SET_CHECK' as 'SET_CHECK',
  payload: {check},
})

export const removeCheck = (checkID: string) => ({
  type: 'REMOVE_CHECK' as 'REMOVE_CHECK',
  payload: {checkID},
})

export const setCurrentCheck = (
  status: RemoteDataState,
  check: Partial<Check>
) => ({
  type: 'SET_CURRENT_CHECK' as 'SET_CURRENT_CHECK',
  payload: {status, check},
})

export const setCurrentCheckStatus = (status: RemoteDataState) => ({
  type: 'SET_CURRENT_CHECK_STATUS' as 'SET_CURRENT_CHECK_STATUS',
  payload: {status},
})

export const updateCurrentCheck = (checkUpdate: Partial<Check>) => ({
  type: 'UPDATE_CURRENT_CHECK' as 'UPDATE_CURRENT_CHECK',
  payload: {status, checkUpdate},
})

export const changeCurrentCheckType = (type: CheckType) => ({
  type: 'CHANGE_CURRENT_CHECK_TYPE' as 'CHANGE_CURRENT_CHECK_TYPE',
  payload: {status, type},
})

export const getChecks = () => async (
  dispatch: Dispatch<Action | NotificationAction>
  // getState: GetState
) => {
  try {
    dispatch(setAllChecks(RemoteDataState.Loading))
    // TODO: use this when its actually implemented
    // const {
    //   orgs: {
    //     org: {id: orgID},
    //   },
    // } = getState()

    // const resp = await api.getChecks({query: {orgID}})

    // if (resp.status !== 200) {
    //   throw new Error(resp.data.message)
    // }

    dispatch(setAllChecks(RemoteDataState.Done, []))
  } catch (e) {
    console.error(e)
    dispatch(setAllChecks(RemoteDataState.Error))
    dispatch(notify(copy.getChecksFailed(e.message)))
  }
}

export const getCurrentCheck = (checkID: string) => async (
  dispatch: Dispatch<Action | NotificationAction>
) => {
  try {
    dispatch(setCurrentCheckStatus(RemoteDataState.Loading))

    const resp = await api.getCheck({checkID})

    if (resp.status !== 200) {
      throw new Error(resp.data.message)
    }

    dispatch(setCurrentCheck(RemoteDataState.Done, resp.data))
  } catch (e) {
    console.error(e)
    dispatch(setCurrentCheckStatus(RemoteDataState.Error))
    dispatch(notify(copy.getCheckFailed(e.message)))
  }
}

export const saveCurrentCheck = () => async (
  dispatch: Dispatch<Action | NotificationAction>,
  getState: GetState
) => {
  try {
    const state = getState()
    const {
      checks: {
        current: {check},
      },
      orgs: {
        org: {id: orgID},
      },
    } = state

    const {draftQueries} = getActiveTimeMachine(state)

    const checkWithOrg = {...check, query: draftQueries[0], orgID} as Check

    const resp = check.id
      ? await api.patchCheck({checkID: check.id, data: checkWithOrg})
      : await api.postCheck({data: checkWithOrg})

    if (resp.status === 201 || resp.status === 200) {
      dispatch(setCheck(resp.data))
    } else {
      throw new Error(resp.data.message)
    }
  } catch (e) {
    console.error(e)
    dispatch(notify(copy.createCheckFailed(e.message)))
  }
}

export const updateCheck = (check: Partial<Check>) => async (
  dispatch: Dispatch<Action | NotificationAction>
) => {
  try {
    const resp = await api.patchCheck({checkID: check.id, data: check as Check})

    if (resp.status === 200) {
      dispatch(setCheck(resp.data))
    } else {
      throw new Error(resp.data.message)
    }

    dispatch(setCheck(resp.data))
  } catch (e) {
    console.error(e)
    dispatch(notify(copy.updateCheckFailed(e.message)))
  }
}

export const deleteCheck = (checkID: string) => async (
  dispatch: Dispatch<Action | NotificationAction>
) => {
  try {
    const resp = await api.deleteCheck({checkID})

    if (resp.status === 204) {
      dispatch(removeCheck(checkID))
    } else {
      throw new Error(resp.data.message)
    }

    dispatch(removeCheck(checkID))
  } catch (e) {
    console.error(e)
    dispatch(notify(copy.deleteCheckFailed(e.message)))
  }
}
