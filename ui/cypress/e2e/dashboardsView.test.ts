import {Organization} from '../../src/types'

const dispatch = action =>
  cy
    .window()
    .its('store')
    .invoke('dispatch', action)

describe('Dashboard', () => {
  beforeEach(() => {
    cy.flush()

    cy.signin().then(({body}) => {
      cy.wrap(body.org).as('org')
    })

    cy.fixture('routes').then(({orgs}) => {
      cy.get('@org').then(({id: orgID}: Organization) => {
        cy.visit(`${orgs}/${orgID}/dashboards`)
      })
    })
  })

  it('can edit a dashboards name', () => {
    cy.get('@org').then(({id: orgID}: Organization) => {
      cy.createDashboard(orgID).then(({body}) => {
        cy.fixture('routes').then(({orgs}) => {
          cy.visit(`${orgs}/${orgID}/dashboards/${body.id}`)
        })
      })
    })

    const newName = 'new 🅱️ashboard'

    cy.get('.renamable-page-title--title').click()
    cy.get('.cf-input-field')
      .type(newName)
      .type('{enter}')

    cy.fixture('routes').then(({orgs}) => {
      cy.get('@org').then(({id: orgID}: Organization) => {
        cy.visit(`${orgs}/${orgID}/dashboards`)
      })
    })

    cy.getByTestID('dashboard-card').should('contain', newName)
  })

  it('can create a cell', () => {
    cy.get('@org').then(({id: orgID}: Organization) => {
      cy.createDashboard(orgID).then(({body}) => {
        cy.fixture('routes').then(({orgs}) => {
          cy.visit(`${orgs}/${orgID}/dashboards/${body.id}`)
        })
      })
    })

    cy.getByTestID('add-cell--button').click()
    cy.getByTestID('save-cell--button').click()
    cy.getByTestID('cell--view-empty').should('have.length', 1)
  })

  const getSelectedVariable = (contextID: string, variableID: string) => win =>
    win.store.getState().variables.values[contextID].values[variableID]
      .selectedValue

  it('can manage variable state with a lot of pointing and clicking', () => {
    cy.get('@org').then(({id: orgID}: Organization) => {
      cy.createDashboard(orgID).then(({body: dashboard}) => {
        cy.createMapVariable(orgID).then(({body: variable}) => {
          cy.fixture('routes').then(({orgs}) => {
            cy.visit(`${orgs}/${orgID}/dashboards/${dashboard.id}`)
          })
          const [firstKey, secondKey] = Object.keys(variable.arguments.values)

          // add cell with variable in its query
          cy.getByTestID('add-cell--button').click()
          cy.getByTestID('switch-to-script-editor').click()
          cy.getByTestID('toolbar-tab').click()
          cy.get('.variables-toolbar--label').click()
          cy.getByTestID('save-cell--button').click()

          // selected value in dashboard is 1st value
          cy.getByTestID('variable-dropdown').should('contain', firstKey)
          cy.window()
            .pipe(getSelectedVariable(dashboard.id, variable.id))
            .should('equal', firstKey)

          // select 2nd value in dashboard
          cy.getByTestID('variable-dropdown--button').click()
          cy.get(`#${secondKey}`).click()

          // selected value in dashboard is 2nd value
          cy.getByTestID('variable-dropdown').should('contain', secondKey)
          cy.window()
            .pipe(getSelectedVariable(dashboard.id, variable.id))
            .should('equal', secondKey)

          // open CEO
          cy.getByTestID('cell-context-menu--edit').click()
          cy.getByTestID('cell-context-menu-item--configure').click()

          // selected value in cell context is 2nd value
          cy.window()
            .pipe(getSelectedVariable('veo', variable.id))
            .should('equal', secondKey)

          // select 1st value in cell
          dispatch({
            type: 'SELECT_VARIABLE_VALUE',
            payload: {
              contextID: 'veo',
              variableID: variable.id,
              selectedValue: firstKey,
            },
          })

          // selected value in cell context is 1st value
          cy.window()
            .pipe(getSelectedVariable('veo', variable.id))
            .should('equal', firstKey)

          // save cell
          cy.getByTestID('save-cell--button').click()

          // selected value in dashboard is 1st value
          cy.getByTestID('variable-dropdown').should('contain', secondKey)
          cy.window()
            .pipe(getSelectedVariable(dashboard.id, variable.id))
            .should('equal', secondKey)
        })
      })
    })
  })

  it('can create a view through the API', () => {
    cy.get('@org').then(({id: orgID}: Organization) => {
      cy.createDashWithViewAndVar(orgID).then(() => {
        cy.fixture('routes').then(({orgs}) => {
          cy.visit(`${orgs}/${orgID}/dashboards`)
          cy.getByTestID('dashboard-card--name').click()
          cy.get('.cell--view').should('have.length', 1)
        })
      })
    })
  })
})
