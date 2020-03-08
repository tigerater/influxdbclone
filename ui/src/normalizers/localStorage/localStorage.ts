// Libraries
import {get} from 'lodash'

// Types
import {LocalStorage} from 'src/types'

// Constants
import {VERSION} from 'src/shared/constants'

// Utils
import {
  normalizeRanges,
  normalizeApp,
  normalizeOrgs,
} from 'src/normalizers/localStorage'

export const normalizeGetLocalStorage = (state: LocalStorage): LocalStorage => {
  let newState = state

  if (state.ranges) {
    newState = {...newState, ranges: normalizeRanges(state.ranges)}
  }

  const appPersisted = get(newState, 'app.persisted', false)
  if (appPersisted) {
    newState = {
      ...newState,
      app: normalizeApp(newState.app),
    }
  }

  return newState
}

export const normalizeSetLocalStorage = (state: LocalStorage): LocalStorage => {
  const {app, ranges, autoRefresh, variables, userSettings, orgs} = state
  return {
    VERSION,
    variables,
    autoRefresh,
    userSettings,
    app: normalizeApp(app),
    orgs: normalizeOrgs(orgs),
    ranges: normalizeRanges(ranges),
  }
}
