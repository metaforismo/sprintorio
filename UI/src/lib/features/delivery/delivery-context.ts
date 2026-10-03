import type { DeliveryPlan } from '../../types/delivery-plan.ts';

export function deliveryReadiness(plan: DeliveryPlan): string {
	if (plan.test_cases.some((test) => test.status === 'failed')) return 'Changes needed';
	if (plan.test_cases.some((test) => test.status === 'blocked')) return 'Testing blocked';
	if (!plan.test_cases.length) return 'No tests recorded';
	if (plan.test_cases.some((test) => test.status === 'not_run')) return 'Testing pending';
	const briefComplete = [plan.product_name, plan.objective, plan.success_metric].every((value) => value.trim());
	const evidenceComplete = plan.test_cases.every((test) =>
		[test.title, test.steps, test.expected_result, test.evidence].every((value) => value.trim())
	);
	return briefComplete && evidenceComplete && plan.milestones.every((milestone) => milestone.status === 'done')
		? 'Ready for review'
		: 'Delivery work pending';
}

export function deliveryContext(slug: string, projectId: string, version: number, plan: DeliveryPlan) {
	const attention = plan.test_cases.filter((test) => test.status !== 'passed');
	return {
		workspace: slug,
		project_id: projectId,
		version,
		basis: 'saved_manual_verification',
		readiness: deliveryReadiness(plan),
		brief: {
			product: plan.product_name,
			objective: plan.objective,
			success_metric: plan.success_metric,
			target_release: plan.target_release || null
		},
		milestones: { total: plan.milestones.length, done: plan.milestones.filter((item) => item.status === 'done').length },
		tests: {
			total: plan.test_cases.length,
			passed: plan.test_cases.filter((item) => item.status === 'passed').length,
			failed: plan.test_cases.filter((item) => item.status === 'failed').length,
			blocked: plan.test_cases.filter((item) => item.status === 'blocked').length,
			not_run: plan.test_cases.filter((item) => item.status === 'not_run').length,
			attention: attention.slice(0, 10).map(({ id, title, status }) => ({ id, title, status })),
			attention_omitted: Math.max(0, attention.length - 10)
		}
	};
}
