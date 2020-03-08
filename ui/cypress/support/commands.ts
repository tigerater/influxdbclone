export const signin = (): Cypress.Chainable<Cypress.Response> => {
  return cy.fixture('user').then(({username, password}) => {
    return cy.setupUser().then(body => {
      return cy
        .request({
          method: 'POST',
          url: '/api/v2/signin',
          auth: {user: username, pass: password},
        })
        .then(() => {
          return cy.wrap(body)
        })
    })
  })
}

export const createDashboard = (
  orgID?: string,
  name: string = 'test dashboard'
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/dashboards',
    body: {
      name,
      orgID,
    },
  })
}

export const createCell = (
  dbID: string,
  dims: {x: number; y: number; height: number; width: number} = {
    x: 0,
    y: 0,
    height: 4,
    width: 4,
  },
  name?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: `/api/v2/dashboards/${dbID}/cells`,
    body: {
      x: dims.x,
      y: dims.y,
      h: dims.height,
      w: dims.width,
      name: name,
    },
  })
}

export const createDashboardTemplate = (
  orgID?: string,
  name: string = 'Bashboard'
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/documents/templates',
    body: {
      content: {
        data: {
          attributes: {name, description: ''},
          relationships: {
            label: {data: []},
            cell: {data: []},
            variable: {data: []},
          },
          type: 'dashboard',
        },
        included: [],
      },
      labels: [],
      meta: {
        description: `template created from dashboard: ${name}`,
        version: '1',
        name: `${name}-Template`,
      },
      orgID,
    },
  })
}

export const createOrg = (): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/orgs',
    body: {
      name: 'test org',
    },
  })
}

export const createBucket = (
  orgID?: string,
  organization?: string,
  bucketName?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/buckets',
    body: {
      name: bucketName,
      orgID,
      organization,
      retentionRules: [],
    },
  })
}

export const createTask = (
  token: string,
  orgID?: string,
  name: string = '🦄ask'
): Cypress.Chainable<Cypress.Response> => {
  const flux = `option task = {
    name: "${name}",
    every: 24h,
    offset: 20m
  }
  from(bucket: "defbuck")
        |> range(start: -2m)`

  return cy.request({
    method: 'POST',
    url: '/api/v2/tasks',
    body: {
      flux,
      orgID,
      token,
    },
  })
}

export const createVariable = (
  orgID?: string
): Cypress.Chainable<Cypress.Response> => {
  const argumentsObj = {
    type: 'query',
    values: {
      language: 'flux',
      query: `filter(fn: (r) => r._field == "cpu")`,
    },
  }

  return cy.request({
    method: 'POST',
    url: '/api/v2/variables',
    body: {
      name: 'Little Variable',
      orgID,
      arguments: argumentsObj,
    },
  })
}

export const createLabel = (
  name?: string,
  orgID?: string,
  properties: {description: string; color: string} = {
    description: `test ${name}`,
    color: '#ff0054',
  }
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/labels',
    body: {
      name,
      orgID,
      properties: properties,
    },
  })
}

export const createAndAddLabel = (
  resource: string,
  orgID: string = '',
  resourceID: string,
  name?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy
    .request({
      method: 'POST',
      url: '/api/v2/labels',
      body: {
        name,
        orgID,
        properties: {
          description: `test ${name}`,
          color: '#ff00ff',
        },
      },
    })
    .then(({body}) => {
      return addResourceLabel(resource, resourceID, body.label.id)
    })
}

export const addResourceLabel = (
  resource: string,
  resourceID: string,
  labelID: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: `/api/v2/${resource}/${resourceID}/labels`,
    body: {labelID},
  })
}

export const createSource = (
  orgID?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/sources',
    body: {
      name: 'defsource',
      default: true,
      orgID,
      type: 'self',
    },
  })
}

export const createScraper = (
  scraperName?: string,
  url?: string,
  type?: string,
  orgID?: string,
  bucketID?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/scrapers',
    body: {
      name: scraperName,
      type,
      url,
      orgID,
      bucketID,
    },
  })
}

export const createTelegraf = (
  name?: string,
  description?: string,
  orgID?: string
): Cypress.Chainable<Cypress.Response> => {
  return cy.request({
    method: 'POST',
    url: '/api/v2/telegrafs',
    body: {
      name,
      description,
      agent: {collectionInterval: 10000},
      plugins: [],
      orgID,
    },
  })
}

/*
[{action: 'write', resource: {type: 'views'}},
      {action: 'write', resource: {type: 'documents'}},
      {action: 'write', resource: {type: 'dashboards'}},
      {action: 'write', resource: {type: 'buckets'}}]}
 */

export const createToken = (
  orgId: string,
  description: string,
  status: string,
  permissions: object[]
): Cypress.Chainable<Cypress.Response> => {
  return cy.request('POST', 'api/v2/authorizations', {
    orgID: orgId,
    description: description,
    status: status,
    permissions: permissions,
  })
}

// TODO: have to go through setup because we cannot create a user w/ a password via the user API
export const setupUser = (): Cypress.Chainable<Cypress.Response> => {
  return cy.fixture('user').then(({username, password, org, bucket}) => {
    return cy.request({
      method: 'POST',
      url: '/api/v2/setup',
      body: {username, password, org, bucket},
    })
  })
}

export const flush = () => {
  cy.request({
    method: 'GET',
    url: '/debug/flush',
  })
}

export const writeData = (
  lines: string[],
  chunkSize: number = 100
): Cypress.Chainable<Cypress.Response> => {
  return cy.fixture('user').then(({org, bucket}) => {
    let chunk: string[]
    let chunkCt: number = 0
    while (chunkCt < lines.length) {
      chunk =
        chunkCt + chunkSize <= lines.length
          ? lines.slice(chunkCt, chunkCt + chunkSize - 1)
          : lines.slice(chunkCt, chunkCt + (chunkSize % lines.length))
      cy.request({
        method: 'POST',
        url: '/api/v2/write?org=' + org + '&bucket=' + bucket,
        body: chunk.join('\n'),
      })
      chunkCt += chunkSize
      chunk = []
    }
  })
}

// DOM node getters
export const getByTestID = (dataTest: string): Cypress.Chainable => {
  return cy.get(`[data-testid="${dataTest}"]`)
}

export const getByTestIDSubStr = (dataTest: string): Cypress.Chainable => {
  return cy.get(`[data-testid*="${dataTest}"]`)
}

export const getByInputName = (name: string): Cypress.Chainable => {
  return cy.get(`input[name=${name}]`)
}

export const getByTitle = (name: string): Cypress.Chainable => {
  return cy.get(`[title="${name}"]`)
}

// custom assertions

// fluxEqual strips flux scripts of whitespace and newlines to make the
// strings easier to match by the human eye during testing
export const fluxEqual = (s1: string, s2: string): Cypress.Chainable => {
  // remove new lines and spaces
  const strip = (s: string) => s.replace(/(\r\n|\n|\r| +)/g, '')
  const strip1 = strip(s1)
  const strip2 = strip(s2)

  cy.log('comparing strings: ')
  cy.log(strip1)
  cy.log(strip2)

  return cy.wrap(strip1 === strip2)
}

// assertions
Cypress.Commands.add('fluxEqual', fluxEqual)

// getters
Cypress.Commands.add('getByTestID', getByTestID)
Cypress.Commands.add('getByInputName', getByInputName)
Cypress.Commands.add('getByTitle', getByTitle)
Cypress.Commands.add('getByTestIDSubStr', getByTestIDSubStr)

// auth flow
Cypress.Commands.add('signin', signin)

// setup
Cypress.Commands.add('setupUser', setupUser)

// dashboards
Cypress.Commands.add('createDashboard', createDashboard)
Cypress.Commands.add('createDashboardTemplate', createDashboardTemplate)
Cypress.Commands.add('createCell', createCell)

// orgs
Cypress.Commands.add('createOrg', createOrg)

// buckets
Cypress.Commands.add('createBucket', createBucket)

// scrapers
Cypress.Commands.add('createScraper', createScraper)

// telegrafs
Cypress.Commands.add('createTelegraf', createTelegraf)

// general
Cypress.Commands.add('flush', flush)

// tasks
Cypress.Commands.add('createTask', createTask)

//Tokems
Cypress.Commands.add('createToken', createToken)

// variables
Cypress.Commands.add('createVariable', createVariable)

// Labels
Cypress.Commands.add('createLabel', createLabel)
Cypress.Commands.add('createAndAddLabel', createAndAddLabel)

//Test
Cypress.Commands.add('writeData', writeData)
