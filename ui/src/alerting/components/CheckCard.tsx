// Libraries
import React, {FunctionComponent} from 'react'
import {connect} from 'react-redux'
import {withRouter, WithRouterProps} from 'react-router'

// Components
import {SlideToggle, ComponentSize, ResourceCard} from '@influxdata/clockface'
import CheckCardContext from 'src/alerting/components/CheckCardContext'
import InlineLabels from 'src/shared/components/inlineLabels/InlineLabels'

// Constants
import {DEFAULT_CHECK_NAME} from 'src/alerting/constants'
import {
  SEARCH_QUERY_PARAM,
  HISTORY_TYPE_QUERY_PARAM,
} from 'src/alerting/constants/history'

// Actions and Selectors
import {
  updateCheck,
  deleteCheck,
  addCheckLabel,
  deleteCheckLabel,
  cloneCheck,
} from 'src/alerting/actions/checks'
import {createLabel as createLabelAsync} from 'src/labels/actions'
import {viewableLabels} from 'src/labels/selectors'
import {notify} from 'src/shared/actions/notifications'
import {updateCheckFailed} from 'src/shared/copy/notifications'

// Types
import {Check, Label, AppState, AlertHistoryType} from 'src/types'

// Utilities
import {relativeTimestampFormatter} from 'src/shared/utils/relativeTimestampFormatter'

interface DispatchProps {
  updateCheck: typeof updateCheck
  deleteCheck: typeof deleteCheck
  onAddCheckLabel: typeof addCheckLabel
  onRemoveCheckLabel: typeof deleteCheckLabel
  onCreateLabel: typeof createLabelAsync
  onCloneCheck: typeof cloneCheck
  onNotify: typeof notify
}

interface StateProps {
  labels: Label[]
}

interface OwnProps {
  check: Check
}

type Props = OwnProps & DispatchProps & WithRouterProps & StateProps

const CheckCard: FunctionComponent<Props> = ({
  onRemoveCheckLabel,
  onAddCheckLabel,
  onCreateLabel,
  onCloneCheck,
  onNotify,
  check,
  updateCheck,
  deleteCheck,
  params: {orgID},
  labels,
  router,
}) => {
  const onUpdateName = async (name: string) => {
    try {
      await updateCheck({...check, name})
    } catch (e) {
      onNotify(updateCheckFailed(e.message))
    }
  }

  const onUpdateDescription = async (description: string) => {
    try {
      await updateCheck({...check, description})
    } catch (e) {
      onNotify(updateCheckFailed(e.message))
    }
  }

  const onDelete = () => {
    deleteCheck(check.id)
  }

  const onClone = () => {
    onCloneCheck(check)
  }

  const onToggle = async () => {
    const status = check.status === 'active' ? 'inactive' : 'active'

    try {
      await updateCheck({...check, status})
    } catch (e) {
      onNotify(updateCheckFailed(e.message))
    }
  }

  const onCheckClick = () => {
    router.push(`/orgs/${orgID}/alerting/checks/${check.id}/edit`)
  }

  const onView = () => {
    const historyType: AlertHistoryType = 'statuses'

    const queryParams = new URLSearchParams({
      [HISTORY_TYPE_QUERY_PARAM]: historyType,
      [SEARCH_QUERY_PARAM]: `"checkID" == "${check.id}"`,
    })

    router.push(`/orgs/${orgID}/alert-history?${queryParams}`)
  }

  const handleAddCheckLabel = (label: Label) => {
    onAddCheckLabel(check.id, label)
  }

  const handleRemoveCheckLabel = (label: Label) => {
    onRemoveCheckLabel(check.id, label)
  }

  const handleCreateLabel = async (label: Label) => {
    await onCreateLabel(label.name, label.properties)
  }

  return (
    <ResourceCard
      key={`check-id--${check.id}`}
      testID="check-card"
      name={
        <ResourceCard.EditableName
          onUpdate={onUpdateName}
          onClick={onCheckClick}
          name={check.name}
          noNameString={DEFAULT_CHECK_NAME}
          testID="check-card--name"
          buttonTestID="check-card--name-button"
          inputTestID="check-card--input"
        />
      }
      toggle={
        <SlideToggle
          active={check.status === 'active'}
          size={ComponentSize.ExtraSmall}
          onChange={onToggle}
          testID="check-card--slide-toggle"
        />
      }
      description={
        <ResourceCard.EditableDescription
          onUpdate={onUpdateDescription}
          description={check.description}
          placeholder={`Describe ${check.name}`}
        />
      }
      labels={
        <InlineLabels
          selectedLabels={check.labels as Label[]}
          labels={labels}
          onAddLabel={handleAddCheckLabel}
          onRemoveLabel={handleRemoveCheckLabel}
          onCreateLabel={handleCreateLabel}
        />
      }
      disabled={check.status === 'inactive'}
      contextMenu={
        <CheckCardContext
          onView={onView}
          onDelete={onDelete}
          onClone={onClone}
        />
      }
      metaData={[
        <>{relativeTimestampFormatter(check.updatedAt, 'Last updated ')}</>,
      ]}
    />
  )
}

const mdtp: DispatchProps = {
  updateCheck: updateCheck,
  deleteCheck: deleteCheck,
  onAddCheckLabel: addCheckLabel,
  onCreateLabel: createLabelAsync,
  onRemoveCheckLabel: deleteCheckLabel,
  onCloneCheck: cloneCheck,
  onNotify: notify,
}

const mstp = ({labels}: AppState): StateProps => {
  return {
    labels: viewableLabels(labels.list),
  }
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(withRouter(CheckCard))
