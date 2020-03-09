// Libraries
import {normalize} from 'normalizr'
import {Dispatch} from 'react'

// API
import * as authAPI from 'src/authorizations/apis'
import * as api from 'src/client'

// Schemas
import * as schemas from 'src/schemas'

// Actions
import {
  Action,
  addAuthorization,
  setAuthorizations,
  removeAuthorization,
} from 'src/authorizations/actions/creators'
import {notify} from 'src/shared/actions/notifications'

// Constants
import {
  authorizationsGetFailed,
  authorizationCreateFailed,
  authorizationUpdateFailed,
  authorizationDeleteFailed,
  authorizationCreateSuccess,
  authorizationDeleteSuccess,
  authorizationUpdateSuccess,
} from 'src/shared/copy/notifications'

// Types
import {
  RemoteDataState,
  GetState,
  NotificationAction,
  Authorization,
  AuthEntities,
} from 'src/types'

// Selectors
import {getOrg} from 'src/organizations/selectors'

type GetAuthorizations = (
  dispatch: Dispatch<Action | NotificationAction>,
  getState: GetState
) => Promise<void>
export const getAuthorizations = () => async (
  dispatch: Dispatch<Action | NotificationAction>,
  getState: GetState
) => {
  try {
    dispatch(setAuthorizations(RemoteDataState.Loading))
    const org = getOrg(getState())
    const resp = await api.getAuthorizations({query: {orgID: org.id}})

    if (resp.status !== 200) {
      throw new Error(resp.data.message)
    }

    const auths = normalize<Authorization, AuthEntities, string[]>(
      resp.data.authorizations,
      schemas.arrayOfAuths
    )

    dispatch(setAuthorizations(RemoteDataState.Done, auths))
  } catch (error) {
    console.error(error)
    dispatch(setAuthorizations(RemoteDataState.Error))
    dispatch(notify(authorizationsGetFailed()))
  }
}

export const getAuthorization = async (authID: string) => {
  try {
    const resp = await api.getAuthorization({authID})

    if (resp.status !== 200) {
      throw new Error(resp.data.message)
    }

    return resp.data
  } catch (error) {
    console.error(error)
  }
}

export const createAuthorization = (auth: Authorization) => async (
  dispatch: Dispatch<Action | NotificationAction>
) => {
  try {
    const resp = await authAPI.createAuthorization(auth)

    const newAuth = normalize<Authorization, AuthEntities, string>(
      resp,
      schemas.auth
    )

    dispatch(addAuthorization(newAuth))
    dispatch(notify(authorizationCreateSuccess()))
  } catch (error) {
    console.error(error.message)
    dispatch(notify(authorizationCreateFailed()))
    throw error
  }
}

export const updateAuthorization = (authorization: Authorization) => async (
  dispatch: Dispatch<Action | NotificationAction | GetAuthorizations>
) => {
  try {
    const resp = await api.patchAuthorization({
      authID: authorization.id,
      data: authorization,
    })

    if (resp.status !== 200) {
      throw new Error(resp.data.message)
    }

    dispatch(getAuthorizations())
    dispatch(notify(authorizationUpdateSuccess()))
  } catch (e) {
    console.error(e)
    dispatch(notify(authorizationUpdateFailed(authorization.id)))
  }
}

export const deleteAuthorization = (id: string, name: string = '') => async (
  dispatch: Dispatch<Action | NotificationAction>
) => {
  try {
    const resp = await api.deleteAuthorization({authID: id})

    if (resp.status !== 204) {
      throw new Error(resp.data.message)
    }

    dispatch(removeAuthorization(id))
    dispatch(notify(authorizationDeleteSuccess()))
  } catch (e) {
    console.error(e)
    dispatch(notify(authorizationDeleteFailed(name)))
  }
}
