// Libraries
import React, {Component, ChangeEvent, createRef} from 'react'
import {connect} from 'react-redux'
import _ from 'lodash'

// Components
import {SquareButton, IconFont, ComponentColor} from '@influxdata/clockface'
import InlineLabelPopover from 'src/shared/components/inlineLabels/InlineLabelPopover'
import CreateLabelOverlay from 'src/labels/components/CreateLabelOverlay'

// Utils
import {validateLabelUniqueness} from 'src/labels/utils'

// Types
import {Label, RemoteDataState} from 'src/types'
import {OverlayState} from 'src/types/overlay'
import {createLabel} from 'src/labels/actions/thunks'

// Constants
export const ADD_NEW_LABEL_ITEM_ID = 'add-new-label'
export const ADD_NEW_LABEL_LABEL: Label = {
  id: ADD_NEW_LABEL_ITEM_ID,
  name: '',
  properties: {
    color: '#000000',
    description: '',
  },
  status: RemoteDataState.NotStarted,
}

import {ErrorHandling} from 'src/shared/decorators/errors'

interface DispatchProps {
  onCreateLabel: typeof createLabel
}

interface StateProps {}

interface OwnProps {
  selectedLabels: Label[]
  labels: Label[]
  onAddLabel: (label: Label) => void
}

type Props = DispatchProps & StateProps & OwnProps

interface State {
  searchTerm: string
  isPopoverVisible: boolean
  selectedItemID: string
  isCreatingLabel: OverlayState
}

@ErrorHandling
class InlineLabelsEditor extends Component<Props, State> {
  private popoverTrigger = createRef<HTMLDivElement>()

  constructor(props: Props) {
    super(props)

    this.state = {
      selectedItemID: null,
      searchTerm: '',
      isPopoverVisible: false,
      isCreatingLabel: OverlayState.Closed,
    }
  }

  public render() {
    const {isCreatingLabel, searchTerm} = this.state

    return (
      <>
        <div className="inline-labels--editor">
          <div className="inline-labels--add-wrapper" ref={this.popoverTrigger}>
            <div className="inline-labels--add">
              <SquareButton
                color={ComponentColor.Secondary}
                titleText="Add labels"
                icon={IconFont.Plus}
                testID="inline-labels--add"
              />
            </div>
            {this.noLabelsIndicator}
          </div>
          {this.popover}
        </div>
        <CreateLabelOverlay
          isVisible={isCreatingLabel === OverlayState.Open}
          onDismiss={this.handleStopCreatingLabel}
          overrideDefaultName={searchTerm}
          onCreateLabel={this.handleCreateLabel}
          onNameValidation={this.handleEnsureUniqueLabelName}
        />
      </>
    )
  }

  private get popover(): JSX.Element {
    const {labels, selectedLabels} = this.props
    const {searchTerm, selectedItemID} = this.state

    const labelsUsed =
      labels.length > 0 && labels.length === selectedLabels.length

    return (
      <InlineLabelPopover
        searchTerm={searchTerm}
        triggerRef={this.popoverTrigger}
        selectedItemID={selectedItemID}
        onUpdateSelectedItemID={this.handleUpdateSelectedItemID}
        allLabelsUsed={labelsUsed}
        onStartCreatingLabel={this.handleStartCreatingLabel}
        onInputChange={this.handleInputChange}
        filteredLabels={this.filterLabels(searchTerm)}
        onAddLabel={this.handleAddLabel}
      />
    )
  }

  private get noLabelsIndicator(): JSX.Element {
    const {selectedLabels} = this.props

    if (selectedLabels.length) {
      return
    }

    return (
      <div
        className="cf-label cf-label--xs cf-label--colorless"
        onClick={this.handleShowPopover}
        data-testid="inline-labels--empty"
      >
        <span className="cf-label--name">Add a label</span>
      </div>
    )
  }

  private handleAddLabel = async (labelID: string) => {
    const {onAddLabel, labels} = this.props

    const label = labels.find(label => label.id === labelID)

    if (label) {
      this.selectAvailableItem()
      await onAddLabel(label)
    }
  }

  private selectAvailableItem = (): void => {
    const {searchTerm} = this.state

    const filteredLabels = this.filterLabels(searchTerm)

    if (filteredLabels.length) {
      this.handleUpdateSelectedItemID(filteredLabels[0].id)
    }
  }

  private handleUpdateSelectedItemID = (selectedItemID: string): void => {
    this.setState({selectedItemID})
  }

  private handleShowPopover = () => {
    const {availableLabels} = this
    const {isPopoverVisible} = this.state

    if (_.isEmpty(availableLabels)) {
      if (isPopoverVisible) {
        return
      }

      return this.setState({
        isPopoverVisible: true,
        selectedItemID: null,
        searchTerm: '',
      })
    }

    const selectedItemID = this.availableLabels[0].id
    this.setState({isPopoverVisible: true, selectedItemID, searchTerm: ''})
  }

  private handleInputChange = (e: ChangeEvent<HTMLInputElement>): void => {
    const searchTerm = e.target.value
    const filteredLabels = this.filterLabels(searchTerm)
    if (filteredLabels.length) {
      const selectedItemID = filteredLabels[0].id
      this.setState({searchTerm, selectedItemID})
    } else {
      this.setState({searchTerm})
    }
  }

  private filterLabels = (searchTerm: string): Label[] => {
    const filteredLabels = this.availableLabels.filter(label => {
      const lowercaseName = label.name.toLowerCase()
      const lowercaseSearchTerm = searchTerm.toLowerCase()

      return lowercaseName.includes(lowercaseSearchTerm)
    })

    const searchTermHasExactMatch = filteredLabels.reduce(
      (acc: boolean, current: Label) => {
        return acc === true || current.name === searchTerm
      },
      false
    )

    if (!searchTermHasExactMatch && searchTerm) {
      return this.filteredLabelsWithAddButton(filteredLabels)
    }

    return this.filteredLabelsWithoutAddButton(filteredLabels)
  }

  private filteredLabelsWithAddButton = (filteredLabels: Label[]): Label[] => {
    const {searchTerm} = this.state

    const updatedAddButton = {...ADD_NEW_LABEL_LABEL, name: searchTerm}

    const addButton = filteredLabels.find(
      label => label.id === updatedAddButton.id
    )

    if (addButton) {
      return filteredLabels.map(fl => {
        return fl.id === updatedAddButton.id ? updatedAddButton : fl
      })
    }

    return [updatedAddButton, ...filteredLabels]
  }

  private filteredLabelsWithoutAddButton = (
    filteredLabels: Label[]
  ): Label[] => {
    return filteredLabels.filter(label => label.id !== ADD_NEW_LABEL_ITEM_ID)
  }

  private get availableLabels(): Label[] {
    const {selectedLabels, labels} = this.props

    return _.differenceBy(labels, selectedLabels, label => label.name)
  }

  private handleCreateLabel = async (label: Label) => {
    const {onCreateLabel, onAddLabel} = this.props
    const {name, properties} = label

    await onCreateLabel(name, properties)
    const newLabel = this.props.labels.find(l => l.name === label.name)
    onAddLabel(newLabel)
  }

  private handleStartCreatingLabel = (): void => {
    this.setState({isCreatingLabel: OverlayState.Open})
  }

  private handleStopCreatingLabel = (): void => {
    this.setState({isCreatingLabel: OverlayState.Closed, searchTerm: ''})
  }

  private handleEnsureUniqueLabelName = (name: string): string | null => {
    const {labels} = this.props
    const names = labels.map(label => label.name)

    return validateLabelUniqueness(names, name)
  }
}

const mstp = (): StateProps => {
  return {}
}

const mdtp: DispatchProps = {
  onCreateLabel: createLabel,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(InlineLabelsEditor)
