import {get} from 'lodash'
import {ASSET_LIMIT_ERROR_STATUS} from 'src/cloud/constants/index'
import {LimitsState} from 'src/cloud/reducers/limits'
import {LimitStatus} from 'src/cloud/actions/limits'

export const isLimitError = (error): boolean => {
  return get(error, 'response.status', '') === ASSET_LIMIT_ERROR_STATUS
}

export const extractBucketLimits = (limits: LimitsState): LimitStatus => {
  return get(limits, 'buckets.limitStatus')
}

export const extractBucketMax = (limits: LimitsState): number => {
  return get(limits, 'buckets.maxAllowed') || Infinity // if maxAllowed == 0, there are no limits on asset
}

export const extractDashboardLimits = (limits: LimitsState): LimitStatus => {
  return get(limits, 'dashboards.limitStatus')
}

export const extractDashboardMax = (limits: LimitsState): number => {
  return get(limits, 'dashboards.maxAllowed') || Infinity
}

export const extractTaskLimits = (limits: LimitsState): LimitStatus => {
  return get(limits, 'tasks.limitStatus')
}

export const extractTaskMax = (limits: LimitsState): number => {
  return get(limits, 'tasks.maxAllowed') || Infinity
}

export const extractRateLimitStatus = (limits: LimitsState): LimitStatus => {
  const statuses = [
    get(limits, 'rate.writeKBs.limitStatus'),
    get(limits, 'rate.readKBs.limitStatus'),
  ]

  if (statuses.includes(LimitStatus.EXCEEDED)) {
    return LimitStatus.EXCEEDED
  }

  return LimitStatus.OK
}

export const extractRateLimitResourceName = (limits: LimitsState): string => {
  const rateLimitedResources = []

  if (get(limits, 'rate.writeKBs.limitStatus') === LimitStatus.EXCEEDED) {
    rateLimitedResources.push('writes')
  }

  if (get(limits, 'rate.readKBs.limitStatus') === LimitStatus.EXCEEDED) {
    rateLimitedResources.push('reads')
  }

  return rateLimitedResources.join(' and ')
}
