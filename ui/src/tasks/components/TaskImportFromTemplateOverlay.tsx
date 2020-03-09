// Libraries
import React, {PureComponent} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'
import _ from 'lodash'

// Components
import {
  Button,
  ComponentColor,
  ComponentStatus,
  Overlay,
} from '@influxdata/clockface'
import TemplateBrowser from 'src/templates/components/createFromTemplateOverlay/TemplateBrowser'
import TemplateBrowserEmpty from 'src/templates/components/createFromTemplateOverlay/TemplateBrowserEmpty'

// Actions
import {createTaskFromTemplate as createTaskFromTemplateAction} from 'src/tasks/actions'
import {getTemplateByID} from 'src/templates/actions'

// Types
import {
  TemplateSummary,
  Template,
  TemplateType,
  AppState,
  RemoteDataState,
  TaskTemplate,
} from 'src/types'
import GetResources, {ResourceType} from 'src/shared/components/GetResources'

interface StateProps {
  templates: TemplateSummary[]
  templateStatus: RemoteDataState
}

interface DispatchProps {
  createTaskFromTemplate: typeof createTaskFromTemplateAction
}

interface State {
  selectedTemplateSummary: TemplateSummary
  selectedTemplate: Template
}

type Props = DispatchProps & StateProps

class TaskImportFromTemplateOverlay extends PureComponent<
  Props & WithRouterProps,
  State
> {
  constructor(props) {
    super(props)
    this.state = {
      selectedTemplateSummary: null,
      selectedTemplate: null,
    }
  }

  render() {
    return (
      <GetResources resource={ResourceType.Templates}>
        <Overlay visible={true}>
          <Overlay.Container maxWidth={900}>
            <Overlay.Header
              title="Create Task from a Template"
              onDismiss={this.onDismiss}
            />
            <Overlay.Body>{this.overlayBody}</Overlay.Body>
            <Overlay.Footer>
              <Button
                text="Cancel"
                onClick={this.onDismiss}
                key="cancel-button"
              />
              <Button
                text="Create Task"
                onClick={this.onSubmit}
                key="submit-button"
                testID="create-task-button"
                color={ComponentColor.Success}
                status={this.submitStatus}
              />
            </Overlay.Footer>
          </Overlay.Container>
        </Overlay>
      </GetResources>
    )
  }

  private get overlayBody(): JSX.Element {
    const {selectedTemplateSummary, selectedTemplate} = this.state
    const {templates} = this.props

    if (!templates.length) {
      return <TemplateBrowserEmpty />
    }

    return (
      <TemplateBrowser
        templates={templates}
        selectedTemplate={selectedTemplate}
        selectedTemplateSummary={selectedTemplateSummary}
        onSelectTemplate={this.handleSelectTemplate}
      />
    )
  }

  private get submitStatus(): ComponentStatus {
    const {selectedTemplate} = this.state

    return selectedTemplate ? ComponentStatus.Default : ComponentStatus.Disabled
  }

  private handleSelectTemplate = async (
    selectedTemplateSummary: TemplateSummary
  ): Promise<void> => {
    const selectedTemplate = await getTemplateByID(selectedTemplateSummary.id)

    this.setState({
      selectedTemplateSummary,
      selectedTemplate,
    })
  }

  private onDismiss = () => {
    const {router} = this.props
    router.goBack()
  }

  private onSubmit = async (): Promise<void> => {
    const {createTaskFromTemplate} = this.props
    const taskTemplate = this.state.selectedTemplate as TaskTemplate

    await createTaskFromTemplate(taskTemplate)
    this.onDismiss()
  }
}

const mstp = ({templates: {items, status}}: AppState): StateProps => {
  const filteredTemplates = items.filter(
    t => !t.meta.type || t.meta.type === TemplateType.Task
  )

  const templates = _.sortBy(filteredTemplates, item =>
    item.meta.name.toLocaleLowerCase()
  )

  return {
    templates,
    templateStatus: status,
  }
}

const mdtp: DispatchProps = {
  createTaskFromTemplate: createTaskFromTemplateAction,
}

export default connect<StateProps>(
  mstp,
  mdtp
)(withRouter(TaskImportFromTemplateOverlay))
