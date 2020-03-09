import React, {PureComponent} from 'react'
import {withRouter, WithRouterProps} from 'react-router'
import {connect} from 'react-redux'

// Components
import ImportOverlay from 'src/shared/components/ImportOverlay'

// Copy
import {invalidJSON} from 'src/shared/copy/notifications'

// Actions
import {
  createTemplate as createTemplateAction,
  getTemplates as getTemplatesAction,
} from 'src/templates/actions'
import {notify as notifyAction} from 'src/shared/actions/notifications'

// Types
import {AppState, Organization} from 'src/types'

interface DispatchProps {
  createTemplate: typeof createTemplateAction
  getTemplates: typeof getTemplatesAction
  notify: typeof notifyAction
}

interface StateProps {
  org: Organization
}

interface OwnProps extends WithRouterProps {
  params: {orgID: string}
}

type Props = DispatchProps & OwnProps & StateProps

class TemplateImportOverlay extends PureComponent<Props> {
  public render() {
    return (
      <ImportOverlay
        onDismissOverlay={this.onDismiss}
        resourceName="Template"
        onSubmit={this.handleImportTemplate}
      />
    )
  }

  private onDismiss = () => {
    const {router} = this.props

    router.goBack()
  }

  private handleImportTemplate = (importString: string) => {
    const {createTemplate, getTemplates, notify} = this.props

    let template
    try {
      template = JSON.parse(importString)
    } catch (error) {
      notify(invalidJSON(error.message))
      return
    }
    createTemplate(template)

    getTemplates()

    this.onDismiss()
  }
}

const mstp = (state: AppState, props: Props): StateProps => {
  const {
    orgs: {items},
  } = state

  const org = items.find(o => o.id === props.params.orgID)

  return {org}
}

const mdtp: DispatchProps = {
  notify: notifyAction,
  createTemplate: createTemplateAction,
  getTemplates: getTemplatesAction,
}

export default connect<StateProps, DispatchProps, Props>(
  mstp,
  mdtp
)(withRouter(TemplateImportOverlay))
