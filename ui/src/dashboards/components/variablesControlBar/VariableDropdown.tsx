// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import _ from 'lodash'

// Components
import {
  Dropdown,
  DropdownMenuTheme,
  ComponentStatus,
} from '@influxdata/clockface'

// Actions
import {selectVariableValue} from 'src/dashboards/actions/index'

// Utils
import {getVariableValuesForDropdown} from 'src/dashboards/selectors'

// Types
import {AppState} from 'src/types'

interface StateProps {
  values: {name: string; value: string}[]
  selectedKey: string
}

interface DispatchProps {
  onSelectValue: (
    contextID: string,
    variableID: string,
    value: string
  ) => Promise<void>
}

interface OwnProps {
  variableID: string
  dashboardID: string
}

type Props = StateProps & DispatchProps & OwnProps

class VariableDropdown extends PureComponent<Props> {
  render() {
    const {selectedKey} = this.props
    const dropdownValues = this.props.values || []

    const dropdownStatus =
      dropdownValues.length === 0
        ? ComponentStatus.Disabled
        : ComponentStatus.Default

    return (
      <div className="variable-dropdown">
        {/* TODO: Add variable description to title attribute when it is ready */}
        <Dropdown
          widthPixels={140}
          className="variable-dropdown--dropdown"
          testID="variable-dropdown"
          button={(active, onClick) => (
            <Dropdown.Button
              active={active}
              onClick={onClick}
              testID="variable-dropdown--button"
              status={dropdownStatus}
            >
              {selectedKey || 'No Values'}
            </Dropdown.Button>
          )}
          menu={onCollapse => (
            <Dropdown.Menu
              onCollapse={onCollapse}
              theme={DropdownMenuTheme.Amethyst}
            >
              {dropdownValues.map(({name}) => (
                /*
                Use key as value since they are unique otherwise 
                multiple selection appear in the dropdown
              */
                <Dropdown.Item
                  key={name}
                  id={name}
                  value={name}
                  onClick={this.handleSelect}
                  selected={name === selectedKey}
                  testID="variable-dropdown--item"
                >
                  {name}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          )}
        />
      </div>
    )
  }

  private handleSelect = (selectedKey: string) => {
    const {dashboardID, variableID, onSelectValue, values} = this.props

    const selection = values.find(v => v.name === selectedKey)
    const selectedValue = !!selection ? selection.value : ''

    onSelectValue(dashboardID, variableID, selectedValue)
  }
}

const mstp = (state: AppState, props: OwnProps): StateProps => {
  const {dashboardID, variableID} = props

  const {selectedKey, list} = getVariableValuesForDropdown(
    state,
    variableID,
    dashboardID
  )

  return {values: list, selectedKey}
}

const mdtp = {
  onSelectValue: selectVariableValue as any,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(VariableDropdown)
