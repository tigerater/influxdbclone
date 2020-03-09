// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {ResourceCard} from '@influxdata/clockface'
import InlineLabels from 'src/shared/components/inlineLabels/InlineLabels'
import VariableContextMenu from 'src/variables/components/VariableContextMenu'

// Types
import {AppState, Label, Variable} from 'src/types'

// Selectors
import {viewableLabels} from 'src/labels/selectors'

// Actions
import {
  addVariableLabelAsync,
  removeVariableLabelAsync,
} from 'src/variables/actions'
import {createLabel as createLabelAsync} from 'src/labels/actions'

interface OwnProps {
  variable: Variable
  onDeleteVariable: (variable: Variable) => void
  onEditVariable: (variable: Variable) => void
  onFilterChange: (searchTerm: string) => void
}

interface StateProps {
  labels: Label[]
}

interface DispatchProps {
  onAddVariableLabel: typeof addVariableLabelAsync
  onRemoveVariableLabel: typeof removeVariableLabelAsync
  onCreateLabel: typeof createLabelAsync
}

type Props = OwnProps & DispatchProps & StateProps

class VariableCard extends PureComponent<Props & WithRouterProps> {
  public render() {
    const {variable, onDeleteVariable} = this.props

    return (
      <ResourceCard
        testID="resource-card"
        labels={this.labels}
        contextMenu={
          <VariableContextMenu
            variable={variable}
            onExport={this.handleExport}
            onRename={this.handleRenameVariable}
            onDelete={onDeleteVariable}
          />
        }
        name={
          <ResourceCard.Name
            onClick={this.handleNameClick}
            name={variable.name}
          />
        }
        metaData={[<>Type: {variable.arguments.type}</>]}
      />
    )
  }

  private handleNameClick = (): void => {
    const {
      variable,
      params: {orgID},
      router,
    } = this.props

    router.push(`/orgs/${orgID}/settings/variables/${variable.id}/edit`)
  }

  private get labels(): JSX.Element {
    const {variable, labels, onFilterChange} = this.props
    const collectorLabels = viewableLabels(variable.labels)

    return (
      <InlineLabels
        selectedLabels={collectorLabels}
        labels={labels}
        onFilterChange={onFilterChange}
        onAddLabel={this.handleAddLabel}
        onRemoveLabel={this.handleRemoveLabel}
        onCreateLabel={this.handleCreateLabel}
      />
    )
  }

  private handleAddLabel = (label: Label): void => {
    const {variable, onAddVariableLabel} = this.props

    onAddVariableLabel(variable.id, label)
  }

  private handleRemoveLabel = (label: Label): void => {
    const {variable, onRemoveVariableLabel} = this.props

    onRemoveVariableLabel(variable.id, label)
  }

  private handleCreateLabel = (label: Label): void => {
    const {name, properties} = label
    this.props.onCreateLabel(name, properties)
  }

  private handleExport = () => {
    const {
      router,
      variable,
      params: {orgID},
    } = this.props
    router.push(`/orgs/${orgID}/settings/variables/${variable.id}/export`)
  }

  private handleRenameVariable = () => {
    const {
      router,
      variable,
      params: {orgID},
    } = this.props

    router.push(`/orgs/${orgID}/settings/variables/${variable.id}/rename`)
  }
}

const mstp = ({labels}: AppState): StateProps => {
  return {
    labels: viewableLabels(labels.list),
  }
}

const mdtp: DispatchProps = {
  onCreateLabel: createLabelAsync,
  onAddVariableLabel: addVariableLabelAsync,
  onRemoveVariableLabel: removeVariableLabelAsync,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<Props>(VariableCard))
