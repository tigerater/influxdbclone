// Libraries
import {produce} from 'immer'

// Types
import {RemoteDataState} from 'src/types'
import {Action} from 'src/telegrafs/actions'
import {ITelegraf as Telegraf} from '@influxdata/influx'

const initialState = (): TelegrafsState => ({
  status: RemoteDataState.NotStarted,
  list: [],
  currentConfig: {status: RemoteDataState.NotStarted, item: ''},
})

export interface TelegrafsState {
  status: RemoteDataState
  list: Telegraf[]
  currentConfig: {status: RemoteDataState; item: string}
}

export const telegrafsReducer = (
  state: TelegrafsState = initialState(),
  action: Action
): TelegrafsState =>
  produce(state, draftState => {
    switch (action.type) {
      case 'SET_TELEGRAFS': {
        const {status, list} = action.payload

        draftState.status = status

        if (list) {
          draftState.list = list
        }

        return
      }

      case 'ADD_TELEGRAF': {
        const {telegraf} = action.payload

        draftState.list.push(telegraf)

        return
      }

      case 'EDIT_TELEGRAF': {
        const {telegraf} = action.payload
        const {list} = draftState

        draftState.list = list.map(l => {
          if (l.id === telegraf.id) {
            return telegraf
          }

          return l
        })

        return
      }

      case 'REMOVE_TELEGRAF': {
        const {id} = action.payload
        const {list} = draftState
        const deleted = list.filter(l => {
          return l.id !== id
        })

        draftState.list = deleted
        return
      }

      case 'SET_CURRENT_CONFIG': {
        const {status, item} = action.payload
        draftState.currentConfig.status = status

        if (item) {
          draftState.currentConfig.item = item
        } else {
          draftState.currentConfig.item = null
        }

        return
      }
    }
  })
