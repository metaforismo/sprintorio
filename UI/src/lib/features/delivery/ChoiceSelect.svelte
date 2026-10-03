<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	let {
		label,
		value,
		options,
		disabled = false,
		onchange
	}: {
		label: string;
		value: string;
		options: { value: string; label: string }[];
		disabled?: boolean;
		onchange: (value: string) => void;
	} = $props();
</script>

<Select.Root type="single" {value} {disabled} onValueChange={(value) => value && onchange(value)}>
	<Select.Trigger aria-label={label} class="min-h-11 w-full"
		>{options.find((option) => option.value === value)?.label ?? label}</Select.Trigger
	>
	<Select.Content
		>{#each options as option}<Select.Item value={option.value} label={option.label}>{option.label}</Select.Item
			>{/each}</Select.Content
	>
</Select.Root>
