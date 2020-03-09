import {Bucket, Permission} from 'src/types'

type PermissionTypes = Permission['resource']['type']

function assertNever(x: never): never {
  throw new Error('Unexpected object: ' + x)
}

const allPermissionTypes: PermissionTypes[] = [
  'authorizations',
  'buckets',
  'checks',
  'dashboards',
  'documents',
  'labels',
  'notificationRules',
  'notificationEndpoints',
  'orgs',
  'secrets',
  'scrapers',
  'sources',
  'tasks',
  'telegrafs',
  'users',
  'variables',
  'views',
]

// The switch statement below will cause a TS error
// if all allowable PermissionTypes generated in the client
// generatedRoutes are not included in the switch statement BUT
// they will need to be added to both the switch statement AND the allPermissionTypes array.
const ensureT = (orgID: string) => (t: PermissionTypes) => {
  switch (t) {
    case 'authorizations':
    case 'buckets':
    case 'checks':
    case 'dashboards':
    case 'documents':
    case 'labels':
    case 'notificationRules':
    case 'notificationEndpoints':
    case 'orgs':
    case 'secrets':
    case 'scrapers':
    case 'sources':
    case 'tasks':
    case 'telegrafs':
    case 'users':
    case 'variables':
    case 'views':
      return [
        {
          action: 'read' as 'read',
          resource: {type: t, orgID},
        },
        {
          action: 'write' as 'write',
          resource: {type: t, orgID},
        },
      ]
    default:
      return assertNever(t)
  }
}

export const allAccessPermissions = (orgID: string): Permission[] => {
  const withOrgID = ensureT(orgID)
  return allPermissionTypes.flatMap(withOrgID)
}

export const specificBucketsPermissions = (
  buckets: Bucket[],
  permission: Permission['action']
): Permission[] => {
  return buckets.map(b => {
    return {
      action: permission,
      resource: {
        type: 'buckets' as 'buckets',
        orgID: b.orgID,
        id: b.id,
      },
    }
  })
}

export const allBucketsPermissions = (
  orgID: string,
  permission: Permission['action']
): Permission[] => {
  return [
    {
      action: permission,
      resource: {type: 'buckets', orgID},
    },
  ]
}

export const selectBucket = (
  bucketName: string,
  selectedBuckets: string[]
): string[] => {
  const isSelected = selectedBuckets.find(n => n === bucketName)

  if (isSelected) {
    return selectedBuckets.filter(n => n !== bucketName)
  }

  return [...selectedBuckets, bucketName]
}

export enum BucketTab {
  AllBuckets = 'All Buckets',
  Scoped = 'Scoped',
}
