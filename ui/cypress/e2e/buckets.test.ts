import {Bucket, Organization} from '../../src/types'

describe('Buckets', () => {
  beforeEach(() => {
    cy.flush()

    cy.signin().then(({body}) => {
      const {
        org: {id},
        bucket,
      } = body
      cy.wrap(body.org).as('org')
      cy.wrap(bucket).as('bucket')
      cy.fixture('routes').then(({orgs}) => {
        cy.visit(`${orgs}/${id}/load-data/buckets`)
      })
    })
  })

  describe('from the buckets index page', () => {
    it('can create a bucket', () => {
      const newBucket = '🅱️ucket'
      cy.getByTestID(`bucket--card--name ${newBucket}`).should('not.exist')

      cy.getByTestID('Create Bucket').click()
      cy.getByTestID('overlay--container').within(() => {
        cy.getByInputName('name').type(newBucket)
        cy.get('.cf-button')
          .contains('Create')
          .click()
      })

      cy.getByTestID(`bucket--card--name ${newBucket}`).should('exist')
    })

    it("can update a bucket's retention rules", () => {
      cy.get<Bucket>('@bucket').then(({name}: Bucket) => {
        cy.getByTestID(`bucket--card--name ${name}`).click()
        cy.getByTestID(`bucket--card--name ${name}`).should(
          'not.contain',
          '7 days'
        )
      })

      cy.getByTestID('retention-intervals--button').click()
      cy.getByTestID('duration-selector--button').click()
      cy.getByTestID('duration-selector--7d').click()

      cy.getByTestID('overlay--container').within(() => {
        cy.contains('Save').click()
      })

      cy.get<Bucket>('@bucket').then(() => {
        cy.getByTestID(`cf-resource-card--meta-item`).should(
          'contain',
          '7 days'
        )
      })
    })

    describe('Searching and Sorting', () => {
      beforeEach(() => {
        cy.get<Organization>('@org').then(({id, name}: Organization) => {
          cy.createBucket(id, name, 'cookie')
          cy.createBucket(id, name, 'bucket2')
        })
      })

      it('Searching buckets', () => {
        cy.getByTestID('search-widget').type('cookie')
        cy.getByTestID('bucket-card').should('have.length', 1)
      })

      it('Sorting by Name', () => {
        cy.getByTestID('name-sorter').click()
        cy.getByTestID('bucket-card')
          .first()
          .contains('defbuck')

        cy.getByTestID('name-sorter').click()
        cy.getByTestID('bucket-card')
          .first()
          .contains('_monitoring')
      })

      it('Sorting by Retention', () => {
        cy.getByTestID('retention-sorter').click()
        cy.getByTestID('bucket-card')
          .first()
          .contains('_tasks')

        cy.getByTestID('retention-sorter').click()
        cy.getByTestID('bucket-card')
          .first()
          .contains('bucket2')
      })
    })

    // Currently producing a false negative
    it.skip('can delete a bucket', () => {
      const bucket1 = 'newbucket1'
      cy.get<Organization>('@org').then(({id, name}: Organization) => {
        cy.createBucket(id, name, bucket1)
      })

      cy.getByTestID(`context-delete-menu ${bucket1}`).click()
      cy.getByTestID(`context-delete-bucket ${bucket1}`).click()

      // normally we would assert for empty state here
      // but we cannot because of the default system buckets
      // since cypress selectors are so fast, that sometimes a bucket
      // that is deleted will be selected before it gets deleted
      cy.wait(10000)

      cy.getByTestID(`bucket--card--name ${bucket1}`).should('not.exist')
    })
  })

  describe('delete with predicate', () => {
    beforeEach(() => {
      cy.getByTestID('bucket-delete-task').click()
      cy.getByTestID('overlay--container').should('have.length', 1)
    })

    it('requires consent to perform delete with predicate', () => {
      // confirm delete is disabled
      cy.getByTestID('confirm-delete-btn').should('be.disabled')
      // checks the consent input
      cy.getByTestID('delete-checkbox').check({force: true})
      // can delete
      cy.getByTestID('confirm-delete-btn')
        .should('not.be.disabled')
        .click()
    })

    it.skip('closes the overlay upon a successful delete with predicate submission', () => {
      cy.getByTestID('delete-checkbox').check({force: true})
      cy.getByTestID('confirm-delete-btn').click()
      cy.getByTestID('overlay--container').should('not.exist')
      cy.getByTestID('notification-success').should('have.length', 1)
    })

    it('should require key-value pairs when deleting predicate with filters', () => {
      // confirm delete is disabled
      cy.getByTestID('add-filter-btn').click()
      // checks the consent input
      cy.getByTestID('delete-checkbox').check({force: true})
      // cannot delete
      cy.getByTestID('confirm-delete-btn').should('be.disabled')

      // should display warnings
      cy.getByTestID('form--element-error').should('have.length', 2)

      cy.getByTestID('key-input').type('mean')
      cy.getByTestID('value-input').type(100)

      cy.getByTestID('confirm-delete-btn')
        .should('not.be.disabled')
        .click()
    })
  })

  describe('Routing directly to the edit overlay', () => {
    it('reroutes to buckets view if bucket does not exist', () => {
      cy.get('@org').then(({id}: Organization) => {
        cy.fixture('routes').then(({orgs}) => {
          const idThatDoesntExist = '261234d1a7f932e4'
          cy.visit(`${orgs}/${id}/load-data/buckets/${idThatDoesntExist}/edit`)
          cy.location('pathname').should(
            'be',
            `${orgs}/${id}/load-data/buckets/`
          )
        })
      })
    })

    it('displays overlay if bucket exists', () => {
      cy.get('@org').then(({id: orgID}: Organization) => {
        cy.fixture('routes').then(({orgs}) => {
          cy.get('@bucket').then(({id: bucketID}: Bucket) => {
            cy.visit(`${orgs}/${orgID}/load-data/buckets/${bucketID}/edit`)
            cy.location('pathname').should(
              'be',
              `${orgs}/${orgID}/load-data/buckets/${bucketID}/edit`
            )
          })
          cy.getByTestID(`overlay`).should('exist')
        })
      })
    })
  })
})
