// Libraries
import React, {PureComponent, MouseEvent} from 'react'
import {connect} from 'react-redux'
import _ from 'lodash'
import {withRouter, WithRouterProps} from 'react-router'
import {
  Button,
  ComponentSize,
  ComponentSpacer,
  FlexDirection,
  JustifyContent,
} from '@influxdata/clockface'

// Components
import {Context} from 'src/clockface'
import {ResourceCard, IconFont} from '@influxdata/clockface'
import InlineLabels from 'src/shared/components/inlineLabels/InlineLabels'

// Actions
import {
  deleteTemplate,
  cloneTemplate,
  updateTemplate,
  createResourceFromTemplate,
  removeTemplateLabelsAsync,
  addTemplateLabelsAsync,
} from 'src/templates/actions'
import {createLabel as createLabelAsync} from 'src/labels/actions'

// Selectors
import {viewableLabels} from 'src/labels/selectors'

// Types
import {TemplateSummary, ILabel} from '@influxdata/influx'
import {ComponentColor} from '@influxdata/clockface'
import {AppState, Organization} from 'src/types'

// Constants
import {DEFAULT_TEMPLATE_NAME} from 'src/templates/constants'

interface OwnProps {
  template: TemplateSummary
  onFilterChange: (searchTerm: string) => void
}

interface DispatchProps {
  onDelete: typeof deleteTemplate
  onClone: typeof cloneTemplate
  onUpdate: typeof updateTemplate
  onCreateFromTemplate: typeof createResourceFromTemplate
  onAddTemplateLabels: typeof addTemplateLabelsAsync
  onRemoveTemplateLabels: typeof removeTemplateLabelsAsync
  onCreateLabel: typeof createLabelAsync
}

interface StateProps {
  labels: ILabel[]
  org: Organization
}

type Props = DispatchProps & OwnProps & StateProps

class TemplateCard extends PureComponent<Props & WithRouterProps> {
  public render() {
    const {template, labels, onFilterChange} = this.props

    return (
      <ResourceCard
        testID="template-card"
        contextMenu={this.contextMenu}
        name={
          <ResourceCard.EditableName
            onClick={this.handleNameClick}
            onUpdate={this.handleUpdateTemplateName}
            name={template.meta.name}
            noNameString={DEFAULT_TEMPLATE_NAME}
            testID="template-card--name"
            buttonTestID="template-card--name-button"
            inputTestID="template-card--input"
          />
        }
        description={this.description}
        labels={
          <InlineLabels
            selectedLabels={template.labels}
            labels={labels}
            onFilterChange={onFilterChange}
            onAddLabel={this.handleAddLabel}
            onRemoveLabel={this.handleRemoveLabel}
            onCreateLabel={this.handleCreateLabel}
          />
        }
        metaData={[this.templateType]}
      />
    )
  }

  private handleUpdateTemplateName = (name: string) => {
    const {template} = this.props

    this.props.onUpdate(template.id, {
      ...template,
      meta: {...template.meta, name},
    })
  }

  private handleUpdateTemplateDescription = (description: string) => {
    const {template} = this.props

    this.props.onUpdate(template.id, {
      ...template,
      meta: {...template.meta, description},
    })
  }

  private get description(): JSX.Element {
    const {template} = this.props
    const description = _.get(template, 'meta.description', '')
    const name = _.get(template, 'meta.name', '')

    return (
      <ResourceCard.EditableDescription
        onUpdate={this.handleUpdateTemplateDescription}
        description={description}
        placeholder={`Describe ${name} Template`}
      />
    )
  }

  private get templateType(): JSX.Element {
    const {template} = this.props

    return (
      <div className="resource-list--meta-item">
        {_.get(template, 'meta.type', '')}
      </div>
    )
  }

  private get contextMenu(): JSX.Element {
    const {
      template: {id},
      onDelete,
    } = this.props
    return (
      <ComponentSpacer
        margin={ComponentSize.Medium}
        direction={FlexDirection.Row}
        justifyContent={JustifyContent.FlexEnd}
      >
        <Button
          text="Create"
          color={ComponentColor.Primary}
          size={ComponentSize.ExtraSmall}
          onClick={this.handleCreate}
        />
        <Context>
          <Context.Menu
            icon={IconFont.Duplicate}
            color={ComponentColor.Secondary}
          >
            <Context.Item label="Clone" action={this.handleClone} value={id} />
          </Context.Menu>
          <Context.Menu
            icon={IconFont.Trash}
            color={ComponentColor.Danger}
            testID="context-delete-menu"
          >
            <Context.Item
              label="Delete"
              action={onDelete}
              value={id}
              testID="context-delete-task"
            />
          </Context.Menu>
        </Context>
      </ComponentSpacer>
    )
  }

  private handleCreate = () => {
    const {onCreateFromTemplate, template} = this.props

    onCreateFromTemplate(template.id)
  }

  private handleClone = () => {
    const {
      template: {id},
      onClone,
    } = this.props
    onClone(id)
  }

  private handleNameClick = (e: MouseEvent): void => {
    e.preventDefault()

    this.handleViewTemplate()
  }

  private handleViewTemplate = () => {
    const {router, template, org} = this.props
    router.push(`/orgs/${org.id}/templates/${template.id}/view`)
  }

  private handleAddLabel = (label: ILabel): void => {
    const {template, onAddTemplateLabels} = this.props

    onAddTemplateLabels(template.id, [label])
  }

  private handleRemoveLabel = (label: ILabel): void => {
    const {template, onRemoveTemplateLabels} = this.props

    onRemoveTemplateLabels(template.id, [label])
  }

  private handleCreateLabel = async (label: ILabel): Promise<void> => {
    await this.props.onCreateLabel(label.name, label.properties)
  }
}

const mstp = ({labels, orgs: {org}}: AppState): StateProps => {
  return {
    org,
    labels: viewableLabels(labels.list),
  }
}

const mdtp: DispatchProps = {
  onDelete: deleteTemplate,
  onClone: cloneTemplate,
  onUpdate: updateTemplate,
  onCreateFromTemplate: createResourceFromTemplate,
  onAddTemplateLabels: addTemplateLabelsAsync,
  onRemoveTemplateLabels: removeTemplateLabelsAsync,
  onCreateLabel: createLabelAsync,
}

export default connect<StateProps, DispatchProps, OwnProps>(
  mstp,
  mdtp
)(withRouter<Props>(TemplateCard))
