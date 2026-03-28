interface RunHistoryFiltersProps {
	query?: string;
	filePath?: string;
	dateFrom?: string;
	dateTo?: string;
	onQueryChange?: (value: string) => void;
	onFilePathChange?: (value: string) => void;
	onDateFromChange?: (value: string) => void;
	onDateToChange?: (value: string) => void;
	onApply?: () => void;
	onReset?: () => void;
}

export function RunHistoryFilters({
	query = "",
	filePath = "",
	dateFrom = "",
	dateTo = "",
	onQueryChange,
	onFilePathChange,
	onDateFromChange,
	onDateToChange,
	onApply,
	onReset,
}: RunHistoryFiltersProps) {
	return (
		<section aria-labelledby="run-history-filters-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
			<div>
				<p className="eyebrow">History filters</p>
				<h3 id="run-history-filters-heading" className="mt-2 font-display text-2xl text-text">
					Search saved runs
				</h3>
			</div>
			<div className="mt-5 grid gap-3 md:grid-cols-2">
				<label className="text-sm text-text">
					<span className="mb-2 block text-text-muted">Keyword</span>
					<input className="w-full rounded-2xl border border-border bg-raised px-3 py-2 text-text" onChange={(event) => onQueryChange?.(event.currentTarget.value)} type="text" value={query} />
				</label>
				<label className="text-sm text-text">
					<span className="mb-2 block text-text-muted">Touched file</span>
					<input className="w-full rounded-2xl border border-border bg-raised px-3 py-2 text-text" onChange={(event) => onFilePathChange?.(event.currentTarget.value)} type="text" value={filePath} />
				</label>
				<label className="text-sm text-text">
					<span className="mb-2 block text-text-muted">From</span>
					<input className="w-full rounded-2xl border border-border bg-raised px-3 py-2 text-text" onChange={(event) => onDateFromChange?.(event.currentTarget.value)} type="date" value={dateFrom} />
				</label>
				<label className="text-sm text-text">
					<span className="mb-2 block text-text-muted">To</span>
					<input className="w-full rounded-2xl border border-border bg-raised px-3 py-2 text-text" onChange={(event) => onDateToChange?.(event.currentTarget.value)} type="date" value={dateTo} />
				</label>
			</div>
			<div className="mt-4 flex flex-wrap gap-3">
				<button className="rounded-full border border-brand-mid px-4 py-2 text-sm text-text" onClick={onApply} type="button">
					Apply filters
				</button>
				<button className="rounded-full border border-border px-4 py-2 text-sm text-text" onClick={onReset} type="button">
					Clear filters
				</button>
			</div>
		</section>
	);
}