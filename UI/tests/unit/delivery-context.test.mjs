import assert from 'node:assert/strict';
import test from 'node:test';
import { deliveryContext, deliveryReadiness } from '../../src/lib/features/delivery/delivery-context.ts';
import { emptyDeliveryPlan } from '../../src/lib/types/delivery-plan.ts';

test('agent summaries keep manual evidence and incomplete definitions separate from release approval', () => {
	const plan = { ...emptyDeliveryPlan(), product_name: 'Portal', objective: 'Track orders', success_metric: 'Find current status', test_cases: [{ id: 't', title: 'Sign in', steps: 'Sign in', expected_result: 'Dashboard', status: 'passed', evidence: '' }] };
	assert.equal(deliveryReadiness(plan), 'Delivery work pending');
	plan.test_cases[0].evidence = 'Observed dashboard';
	assert.equal(deliveryReadiness(plan), 'Ready for review');
	plan.test_cases[0].status = 'blocked';
	const context = deliveryContext('demo', 'p', 7, plan);
	assert.equal(context.version, 7);
	assert.equal(context.basis, 'saved_manual_verification');
	assert.equal(context.readiness, 'Testing blocked');
	assert.equal(context.tests.blocked, 1);
	assert.equal(JSON.stringify(context).includes('Observed dashboard'), false);
});

test('large summaries report omitted work without losing total counts', () => {
	const plan = emptyDeliveryPlan();
	plan.test_cases = Array.from({ length: 20 }, (_, i) => ({ id: `t${i}`, title: `Check ${i}`, steps: '', expected_result: '', evidence: '', status: i < 3 ? 'passed' : 'not_run' }));
	const context = deliveryContext('demo', 'p', 0, plan);
	assert.equal(context.tests.total, 20);
	assert.equal(context.tests.passed, 3);
	assert.equal(context.tests.not_run, 17);
	assert.equal(context.tests.attention.length, 10);
	assert.equal(context.tests.attention_omitted, 7);
	assert.equal(deliveryReadiness(emptyDeliveryPlan()), 'No tests recorded');
});
