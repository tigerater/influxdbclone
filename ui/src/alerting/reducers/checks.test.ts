import checksReducer, {defaultChecksState} from 'src/alerting/reducers/checks'
import {
  setAllChecks,
  setCheck,
  setCurrentCheck,
  removeCheck,
} from 'src/alerting/actions/checks'
import {RemoteDataState} from 'src/types'
import {check1, check2} from 'src/alerting/constants'

describe('checksReducer', () => {
  describe('setAllChecks', () => {
    it('sets list and status properties of state.', () => {
      const initialState = defaultChecksState

      const actual = checksReducer(
        initialState,
        setAllChecks(RemoteDataState.Done, [check1, check2])
      )

      const expected = {
        ...defaultChecksState,
        list: [check1, check2],
        status: RemoteDataState.Done,
      }

      expect(actual).toEqual(expected)
    })
  })
  describe('setCheck', () => {
    it('adds check to list if it is new', () => {
      const initialState = defaultChecksState

      const actual = checksReducer(initialState, setCheck(check2))

      const expected = {
        ...defaultChecksState,
        list: [check2],
      }

      expect(actual).toEqual(expected)
    })
    it('updates check in list if it exists', () => {
      let initialState = defaultChecksState
      initialState.list = [check1]
      const actual = checksReducer(
        initialState,
        setCheck({...check1, name: check2.name})
      )

      const expected = {
        ...defaultChecksState,
        list: [{...check1, name: check2.name}],
      }

      expect(actual).toEqual(expected)
    })
  })
  describe('removeCheck', () => {
    it('removes check from list', () => {
      const initialState = defaultChecksState
      initialState.list = [check1]
      const actual = checksReducer(initialState, removeCheck(check1.id))

      const expected = {
        ...defaultChecksState,
        list: [],
      }

      expect(actual).toEqual(expected)
    })
  })
  describe('setCurrentCheck', () => {
    it('sets current check and status.', () => {
      const initialState = defaultChecksState

      const actual = checksReducer(
        initialState,
        setCurrentCheck(RemoteDataState.Done, check1)
      )

      const expected = {
        ...defaultChecksState,
        current: {status: RemoteDataState.Done, check: check1},
      }

      expect(actual).toEqual(expected)
    })
  })
})
