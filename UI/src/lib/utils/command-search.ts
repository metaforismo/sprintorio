const normalize = (text: string) => text.normalize('NFKD').replace(/\p{M}/gu, '').toLowerCase().trim();

export function rankCommands<T extends { label: string; description?: string }>(items: T[], query: string): T[] {
	const term = normalize(query);
	if (!term) return items;
	return items
		.map((item, index) => {
			const text = normalize(`${item.label} ${item.description ?? ''}`);
			const direct = text.indexOf(term);
			if (direct >= 0) return { item, score: direct, index };
			if (term.split(/\s+/).every((word) => text.includes(word))) return { item, score: 100, index };
			let cursor = 0;
			for (const character of term.replace(/\s/g, '')) {
				cursor = text.indexOf(character, cursor);
				if (cursor < 0) return { item, score: Infinity, index };
				cursor++;
			}
			return { item, score: 200 + cursor, index };
		})
		.filter(({ score }) => Number.isFinite(score))
		.sort((a, b) => a.score - b.score || a.index - b.index)
		.map(({ item }) => item);
}
