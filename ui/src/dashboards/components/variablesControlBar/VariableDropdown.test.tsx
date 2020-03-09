// Libraries
import React from 'react'
import {fireEvent} from 'react-testing-library'

// Components
import VariableDropdown from 'src/dashboards/components/variablesControlBar/VariableDropdown'

// Utils
import {renderWithRedux} from 'src/mockState'
import {AppState} from 'src/types'

const values = {
  def: 'defbuck',
  def2: 'defbuck',
  foo: 'foobuck',
  goo: 'goobuck',
  new: 'newBuck',
}

const setInitialState = (state: AppState) => {
  return {
    ...state,
    orgs: [
      {
        id: '03cbdc8a53a63000',
      },
    ],
    variables: {
      status: 'Done',
      variables: {
        '03cbdc8a53a63000': {
          variable: {
            id: '03cbdc8a53a63000',
            orgID: '03c02466515c1000',
            name: 'map_buckets',
            description: '',
            selected: null,
            arguments: {
              type: 'map',
              values,
            },
            labels: [],
          },
          status: 'Done',
        },
      },
      values: {
        '03c8070355fbd000': {
          status: 'Done',
          values: {
            '03cbdc8a53a63000': {
              valueType: 'string',
              values: Object.values(values),
              selectedValue: 'defbuck',
            },
          },
          order: ['03cbdc8a53a63000'],
        },
      },
    },
  }
}

describe('Dashboards.Components.VariablesControlBar.VariableDropdown', () => {
  describe('if map type', () => {
    it('renders dropdown with keys as dropdown items', () => {
      const {getByTestId, getAllByTestId} = renderWithRedux(
        <VariableDropdown
          variableID="03cbdc8a53a63000"
          dashboardID="03c8070355fbd000"
        />,
        setInitialState
      )

      const dropdownButton = getByTestId('variable-dropdown--button')
      fireEvent.click(dropdownButton)
      const dropdownItems = getAllByTestId('variable-dropdown--item')

      expect(dropdownItems.length).toBe(Object.keys(values).length)
    })
  })
})
