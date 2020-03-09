// Libraries
import React, {FunctionComponent, useState, useEffect} from 'react'
import {connect} from 'react-redux'

// Components
import MatchingRuleCard from 'src/alerting/components/builder/MatchingRuleCard'
import {SpinnerContainer, TechnoSpinner} from '@influxdata/clockface'

// API
import * as api from 'src/client'

//Types
import {NotificationRule, AppState, CheckTagSet} from 'src/types'
import {EmptyState, ComponentSize, RemoteDataState} from '@influxdata/clockface'

interface StateProps {
  tags: CheckTagSet[]
  orgID: string
}

const CheckMatchingRulesCard: FunctionComponent<StateProps> = ({
  orgID,
  tags,
}) => {
  const getMatchingRules = async (): Promise<NotificationRule[]> => {
    const tagsList = tags.map(t => ['tag', `${t.key}:${t.value}`])

    // todo also: get tags from query results

    const resp = await api.getNotificationRules({
      query: [['orgID', orgID], ...tagsList] as any,
    })

    if (resp.status !== 200) {
      setMatchingRules({matchingRules: [], status: RemoteDataState.Error})
      return
    }

    setMatchingRules({
      matchingRules: resp.data.notificationRules,
      status: RemoteDataState.Done,
    })
  }

  const [{matchingRules, status}, setMatchingRules] = useState<{
    matchingRules: NotificationRule[]
    status: RemoteDataState
  }>({matchingRules: [], status: RemoteDataState.NotStarted})

  useEffect(() => {
    setMatchingRules({
      matchingRules,
      status: RemoteDataState.Loading,
    })
    getMatchingRules()
  }, [tags])

  const emptyState = (
    <EmptyState
      size={ComponentSize.Small}
      className="alert-builder--card__empty"
    >
      <EmptyState.Text>
        Notification Rules configured to act on tag sets matching this Alert
        Check will automatically show up here
      </EmptyState.Text>
      <EmptyState.Text>
        Looks like no notification rules match the tag set defined in this Alert
        Check
      </EmptyState.Text>
    </EmptyState>
  )

  if (
    status === RemoteDataState.NotStarted ||
    status === RemoteDataState.Loading
  ) {
    return (
      <SpinnerContainer spinnerComponent={<TechnoSpinner />} loading={status} />
    )
  }

  if (matchingRules.length === 0) {
    return emptyState
  }
  return (
    <>
      {matchingRules.map(r => (
        <MatchingRuleCard key={r.id} rule={r} />
      ))}
    </>
  )
}

const mstp = (state: AppState): StateProps => {
  const {
    orgs: {
      org: {id: orgID},
    },
    alertBuilder: {tags},
  } = state

  return {tags, orgID}
}

export default connect<StateProps, {}, {}>(
  mstp,
  null
)(CheckMatchingRulesCard)
