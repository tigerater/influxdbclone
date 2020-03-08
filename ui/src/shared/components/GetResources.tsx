// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'
import _ from 'lodash'

// Actions
import {getLabels} from 'src/labels/actions'
import {getBuckets} from 'src/buckets/actions'
import {getTelegrafs} from 'src/telegrafs/actions'
import {getVariables} from 'src/variables/actions'
import {getScrapers} from 'src/scrapers/actions'
import {getDashboardsAsync} from 'src/dashboards/actions'
import {getTasks} from 'src/tasks/actions'
import {getAuthorizations} from 'src/authorizations/actions'
import {getTemplates} from 'src/templates/actions'
import {getMembers, getUsers} from 'src/members/actions'
import {getChecks} from 'src/alerting/actions/checks'
import {getNotificationRules} from 'src/alerting/actions/notifications/rules'

// Types
import {AppState} from 'src/types'
import {LabelsState} from 'src/labels/reducers'
import {BucketsState} from 'src/buckets/reducers'
import {TelegrafsState} from 'src/telegrafs/reducers'
import {ScrapersState} from 'src/scrapers/reducers'
import {TasksState} from 'src/tasks/reducers/'
import {DashboardsState} from 'src/dashboards/reducers/dashboards'
import {AuthorizationsState} from 'src/authorizations/reducers'
import {VariablesState} from 'src/variables/reducers'
import {TemplatesState} from 'src/templates/reducers'
import {MembersState, UsersMap} from 'src/members/reducers'
import {ChecksState} from 'src/alerting/reducers/checks'
import {NotificationRulesState} from 'src/alerting/reducers/notifications/rules'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import {
  TechnoSpinner,
  SpinnerContainer,
  RemoteDataState,
} from '@influxdata/clockface'

interface StateProps {
  labels: LabelsState
  buckets: BucketsState
  telegrafs: TelegrafsState
  variables: VariablesState
  scrapers: ScrapersState
  tokens: AuthorizationsState
  dashboards: DashboardsState
  templates: TemplatesState
  tasks: TasksState
  members: MembersState
  users: {status: RemoteDataState; item: UsersMap}
  checks: ChecksState
  rules: NotificationRulesState
}

interface DispatchProps {
  getLabels: typeof getLabels
  getBuckets: typeof getBuckets
  getTelegrafs: typeof getTelegrafs
  getVariables: typeof getVariables
  getScrapers: typeof getScrapers
  getAuthorizations: typeof getAuthorizations
  getDashboards: typeof getDashboardsAsync
  getTasks: typeof getTasks
  getTemplates: typeof getTemplates
  getMembers: typeof getMembers
  getUsers: typeof getUsers
  getChecks: typeof getChecks
  getNotificationRules: typeof getNotificationRules
}

interface PassedProps {
  resource: ResourceTypes
}

type Props = StateProps & DispatchProps & PassedProps

export enum ResourceTypes {
  Labels = 'labels',
  Buckets = 'buckets',
  Telegrafs = 'telegrafs',
  Variables = 'variables',
  Authorizations = 'tokens',
  Scrapers = 'scrapers',
  Dashboards = 'dashboards',
  Tasks = 'tasks',
  Templates = 'templates',
  Members = 'members',
  Users = 'users',
  Checks = 'checks',
  NotificationRules = 'rules',
}

@ErrorHandling
class GetResources extends PureComponent<Props, StateProps> {
  public async componentDidMount() {
    switch (this.props.resource) {
      case ResourceTypes.Dashboards: {
        return await this.props.getDashboards()
      }

      case ResourceTypes.Labels: {
        return await this.props.getLabels()
      }

      case ResourceTypes.Buckets: {
        return await this.props.getBuckets()
      }

      case ResourceTypes.Telegrafs: {
        return await this.props.getTelegrafs()
      }

      case ResourceTypes.Scrapers: {
        return await this.props.getScrapers()
      }

      case ResourceTypes.Variables: {
        return await this.props.getVariables()
      }

      case ResourceTypes.Tasks: {
        return await this.props.getTasks()
      }

      case ResourceTypes.Authorizations: {
        return await this.props.getAuthorizations()
      }

      case ResourceTypes.Templates: {
        return await this.props.getTemplates()
      }

      case ResourceTypes.Members: {
        return await this.props.getMembers()
      }

      case ResourceTypes.Users: {
        return await this.props.getUsers()
      }

      case ResourceTypes.Checks: {
        return await this.props.getChecks()
      }

      case ResourceTypes.NotificationRules: {
        return await this.props.getNotificationRules()
      }

      default: {
        throw new Error('incorrect resource type provided')
      }
    }
  }

  public render() {
    const {resource, children} = this.props

    return (
      <SpinnerContainer
        loading={this.props[resource].status}
        spinnerComponent={<TechnoSpinner />}
      >
        <>{children}</>
      </SpinnerContainer>
    )
  }
}

const mstp = ({
  labels,
  buckets,
  telegrafs,
  variables,
  scrapers,
  tokens,
  dashboards,
  tasks,
  templates,
  members,
  checks,
  rules,
}: AppState): StateProps => {
  return {
    labels,
    buckets,
    telegrafs,
    dashboards,
    variables,
    scrapers,
    tokens,
    tasks,
    templates,
    members,
    users: members.users,
    checks,
    rules,
  }
}

const mdtp = {
  getLabels: getLabels,
  getBuckets: getBuckets,
  getTelegrafs: getTelegrafs,
  getVariables: getVariables,
  getScrapers: getScrapers,
  getAuthorizations: getAuthorizations,
  getDashboards: getDashboardsAsync,
  getTasks: getTasks,
  getTemplates: getTemplates,
  getMembers: getMembers,
  getUsers: getUsers,
  getChecks: getChecks,
  getNotificationRules: getNotificationRules,
}

export default connect<StateProps, DispatchProps, {}>(
  mstp,
  mdtp
)(GetResources)
