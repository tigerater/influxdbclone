// Libraries
import _ from 'lodash'

// API
import {client} from 'src/utils/api'
import * as api from 'src/client'

// Types
import {RemoteDataState, GetState} from 'src/types'
import {AddResourceMemberRequestBody} from '@influxdata/influx'
import {Dispatch} from 'redux-thunk'
import {Member} from 'src/types'

// Actions
import {notify} from 'src/shared/actions/notifications'
import {UsersMap} from 'src/members/reducers'
import {
  memberAddSuccess,
  memberAddFailed,
  memberRemoveSuccess,
  memberRemoveFailed,
} from 'src/shared/copy/notifications'

export type Action = SetMembers | AddMember | RemoveMember | SetUsers

interface SetMembers {
  type: 'SET_MEMBERS'
  payload: {
    status: RemoteDataState
    list: Member[]
  }
}

export const setMembers = (
  status: RemoteDataState,
  list?: Member[]
): SetMembers => ({
  type: 'SET_MEMBERS',
  payload: {status, list},
})

interface AddMember {
  type: 'ADD_MEMBER'
  payload: {
    member: Member
  }
}

export const addMember = (member: Member): AddMember => ({
  type: 'ADD_MEMBER',
  payload: {member},
})

interface RemoveMember {
  type: 'REMOVE_MEMBER'
  payload: {id: string}
}

export const removeMember = (id: string): RemoveMember => ({
  type: 'REMOVE_MEMBER',
  payload: {id},
})

interface SetUsers {
  type: 'SET_USERS'
  payload: {
    status: RemoteDataState
    list: UsersMap
  }
}

export const setUsers = (
  status: RemoteDataState,
  list?: UsersMap
): SetUsers => ({
  type: 'SET_USERS',
  payload: {status, list},
})

export const getMembers = () => async (
  dispatch: Dispatch<Action>,
  getState: GetState
) => {
  try {
    const {
      orgs: {
        org: {id},
      },
    } = getState()
    dispatch(setMembers(RemoteDataState.Loading))

    const [ownersResp, membersResp] = await Promise.all([
      api.getOrgsOwners({orgID: id}),
      api.getOrgsMembers({orgID: id}),
    ])

    if (ownersResp.status !== 200) {
      throw new Error(ownersResp.data.message)
    }

    if (membersResp.status !== 200) {
      throw new Error(membersResp.data.message)
    }

    const owners = ownersResp.data.users

    const members = membersResp.data.users

    const users = [...owners, ...members]

    dispatch(setMembers(RemoteDataState.Done, users))
  } catch (e) {
    console.error(e)
    dispatch(setMembers(RemoteDataState.Error))
  }
}

export const addNewMember = (member: AddResourceMemberRequestBody) => async (
  dispatch: Dispatch<Action>,
  getState: GetState
) => {
  try {
    const {
      orgs: {
        org: {id},
      },
    } = getState()

    const resp = await api.postOrgsMember({orgID: id, data: member})

    if (resp.status !== 201) {
      throw new Error(resp.data.message)
    }

    const newMember = resp.data

    dispatch(addMember(newMember))
    dispatch(notify(memberAddSuccess(member.name)))
  } catch (e) {
    console.error(e)
    const message = _.get(e, 'response.data.message', 'Unknown error')
    dispatch(notify(memberAddFailed(message)))
    throw e
  }
}

export const deleteMember = (member: Member) => async (
  dispatch: Dispatch<Action>,
  getState: GetState
) => {
  try {
    const {
      orgs: {
        org: {id},
      },
    } = getState()

    const resp = await api.deleteOrgsMember({orgID: id, userID: member.id})

    if (resp.status !== 204) {
      throw new Error(resp.data.message)
    }

    dispatch(removeMember(member.id))

    dispatch(notify(memberRemoveSuccess(member.name)))
  } catch (e) {
    console.error(e)
    dispatch(notify(memberRemoveFailed(member.name)))
  }
}

export const getUsers = () => async (
  dispatch: Dispatch<Action>,
  getState: GetState
) => {
  try {
    const {
      members: {list},
    } = getState()

    const apiUsers = await client.users.getAll()
    const allUsers = apiUsers.reduce((acc, u) => _.set(acc, u.id, u), {})
    const users = _.omit(allUsers, list.map(m => m.id))

    dispatch(setUsers(RemoteDataState.Done, users))
  } catch (e) {
    console.error(e)
    dispatch(setMembers(RemoteDataState.Error))
  }
}
