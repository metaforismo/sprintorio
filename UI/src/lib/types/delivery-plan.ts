export type MilestoneStatus = 'planned' | 'in_progress' | 'done';
export type TestStatus = 'not_run' | 'passed' | 'failed' | 'blocked';
export interface DeliveryPlan {
	product_name: string;
	objective: string;
	success_metric: string;
	target_release: string;
	milestones: { id: string; title: string; due_date: string; status: MilestoneStatus }[];
	test_cases: {
		id: string;
		title: string;
		steps: string;
		expected_result: string;
		status: TestStatus;
		evidence: string;
	}[];
}
export interface DeliveryPlanResponse {
	plan: DeliveryPlan;
	version: number;
}
export function emptyDeliveryPlan(): DeliveryPlan {
	return { product_name: '', objective: '', success_metric: '', target_release: '', milestones: [], test_cases: [] };
}
