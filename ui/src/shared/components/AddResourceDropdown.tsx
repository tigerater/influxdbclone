// Libraries
import React, {PureComponent} from 'react'
import _ from 'lodash'

// Components
import {
  IconFont,
  ComponentColor,
  ComponentSize,
  Dropdown,
  ComponentStatus,
} from '@influxdata/clockface'

interface OwnProps {
  onSelectNew: () => void
  onSelectImport: () => void
  onSelectTemplate?: () => void
  resourceName: string
}

interface DefaultProps {
  canImportFromTemplate: boolean
  status: ComponentStatus
  titleText: string
}

type Props = OwnProps & DefaultProps

export default class AddResourceDropdown extends PureComponent<Props> {
  public static defaultProps: DefaultProps = {
    canImportFromTemplate: false,
    status: ComponentStatus.Default,
    titleText: null,
  }

  public render() {
    const {titleText, status} = this.props
    return (
      <Dropdown
        widthPixels={160}
        testID="add-resource-dropdown"
        button={(active, onClick) => (
          <Dropdown.Button
            testID="add-resource-dropdown--button"
            active={active}
            onClick={onClick}
            color={ComponentColor.Primary}
            size={ComponentSize.Small}
            icon={IconFont.Plus}
            status={status}
          >
            {titleText || `Create ${this.props.resourceName}`}
          </Dropdown.Button>
        )}
        menu={onCollapse => (
          <Dropdown.Menu
            onCollapse={onCollapse}
            testID="add-resource-dropdown--menu"
          >
            {this.optionItems}
          </Dropdown.Menu>
        )}
      >
        {this.optionItems}
      </Dropdown>
    )
  }

  private get optionItems(): JSX.Element[] {
    const importOption = this.importOption
    const newOption = this.newOption
    const templateOption = this.templateOption

    const items = [
      <Dropdown.Item
        id={newOption}
        key={newOption}
        onClick={this.handleSelect}
        value={newOption}
        testID="add-resource-dropdown--new"
      >
        {newOption}
      </Dropdown.Item>,
      <Dropdown.Item
        id={importOption}
        key={importOption}
        onClick={this.handleSelect}
        value={importOption}
        testID="add-resource-dropdown--import"
      >
        {importOption}
      </Dropdown.Item>,
    ]

    if (!!this.props.canImportFromTemplate) {
      items.push(
        <Dropdown.Item
          id={templateOption}
          key={templateOption}
          onClick={this.handleSelect}
          value={templateOption}
          testID="add-resource-dropdown--template"
        >
          {templateOption}
        </Dropdown.Item>
      )
    }

    return items
  }

  private get newOption(): string {
    return `New ${this.props.resourceName}`
  }

  private get importOption(): string {
    return `Import ${this.props.resourceName}`
  }

  private get templateOption(): string {
    return `From a Template`
  }

  private handleSelect = (selection: string): void => {
    const {onSelectNew, onSelectImport, onSelectTemplate} = this.props

    if (selection === this.newOption) {
      onSelectNew()
    }
    if (selection === this.importOption) {
      onSelectImport()
    }
    if (selection == this.templateOption) {
      onSelectTemplate()
    }
  }
}
