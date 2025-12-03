import {
	AlertCircle,
	Download,
	MoreHorizontal,
	PlayCircle,
	Trash2,
	XCircle,
} from "lucide-react";
import { memo } from "react";
import { formatBytes, formatRelativeTime, truncateText } from "../../lib/utils";
import { type QueueItem, QueueStatus } from "../../types/api";
import { PathDisplay } from "../ui/PathDisplay";
import { StatusBadge } from "../ui/StatusBadge";

interface QueueTableRowProps {
	item: QueueItem;
	isSelected: boolean;
	isDeletePending: boolean;
	isRetryPending: boolean;
	isCancelPending: boolean;
	onSelectItem: (id: number, checked: boolean) => void;
	onRetry: (id: number) => void;
	onCancel: (id: number) => void;
	onDownload: (id: number) => void;
	onDelete: (id: number) => void;
}

export const QueueTableRow = memo(function QueueTableRow({
	item,
	isSelected,
	isDeletePending,
	isRetryPending,
	isCancelPending,
	onSelectItem,
	onRetry,
	onCancel,
	onDownload,
	onDelete,
}: QueueTableRowProps) {
	return (
		<tr className={`hover ${isSelected ? "bg-base-200" : ""}`}>
			<td>
				<label className="cursor-pointer">
					<input
						type="checkbox"
						className="checkbox"
						checked={isSelected}
						onChange={(e) => onSelectItem(item.id, e.target.checked)}
					/>
				</label>
			</td>
			<td>
				<div className="flex items-center space-x-3">
					<Download className="h-4 w-4 text-primary" />
					<div>
						<div className="font-bold">
							<PathDisplay path={item.nzb_path} maxLength={90} showFileName={true} />
						</div>
						<div className="text-base-content/70 text-sm">ID: {item.id}</div>
					</div>
				</div>
			</td>
			<td>
				<PathDisplay path={item.target_path} maxLength={50} className="text-sm" />
			</td>
			<td>
				{item.category ? (
					<span className="badge badge-outline badge-sm">{item.category}</span>
				) : (
					<span className="text-base-content/50 text-sm">—</span>
				)}
			</td>
			<td>
				{item.file_size ? (
					<span className="text-sm">{formatBytes(item.file_size)}</span>
				) : (
					<span className="text-base-content/50 text-sm">—</span>
				)}
			</td>
			<td>
				<div className="flex flex-col gap-1">
					{item.status === QueueStatus.FAILED && item.error_message ? (
						<div
							className="tooltip tooltip-top"
							data-tip={truncateText(item.error_message, 200)}
						>
							<div className="flex items-center gap-1">
								<StatusBadge status={item.status} />
								<AlertCircle className="h-3 w-3 text-error" />
							</div>
						</div>
					) : item.status === QueueStatus.PROCESSING && item.percentage != null ? (
						<div className="flex items-center gap-2">
							<progress
								className="progress progress-primary w-24"
								value={item.percentage}
								max={100}
							/>
							<span className="text-xs">{item.percentage}%</span>
						</div>
					) : (
						<StatusBadge status={item.status} />
					)}
				</div>
			</td>
			<td>
				<span className={`badge ${item.retry_count > 0 ? "badge-warning" : "badge-ghost"}`}>
					{item.retry_count}
				</span>
			</td>
			<td>
				<span className="text-base-content/70 text-sm">
					{formatRelativeTime(item.updated_at)}
				</span>
			</td>
			<td>
				<div className="dropdown dropdown-end">
					<button tabIndex={0} type="button" className="btn btn-ghost btn-sm">
						<MoreHorizontal className="h-4 w-4" />
					</button>
					<ul className="dropdown-content menu w-48 rounded-box bg-base-100 shadow-lg">
						{(item.status === QueueStatus.PENDING ||
							item.status === QueueStatus.FAILED ||
							item.status === QueueStatus.COMPLETED) && (
							<li>
								<button
									type="button"
									onClick={() => onRetry(item.id)}
									disabled={isRetryPending}
								>
									<PlayCircle className="h-4 w-4" />
									{item.status === QueueStatus.PENDING ? "Process" : "Retry"}
								</button>
							</li>
						)}
						{item.status === QueueStatus.PROCESSING && (
							<li>
								<button
									type="button"
									onClick={() => onCancel(item.id)}
									disabled={isCancelPending}
									className="text-warning"
								>
									<XCircle className="h-4 w-4" />
									Cancel
								</button>
							</li>
						)}
						<li>
							<button type="button" onClick={() => onDownload(item.id)}>
								<Download className="h-4 w-4" />
								Download NZB
							</button>
						</li>
						{item.status !== QueueStatus.PROCESSING && (
							<li>
								<button
									type="button"
									onClick={() => onDelete(item.id)}
									disabled={isDeletePending}
									className="text-error"
								>
									<Trash2 className="h-4 w-4" />
									Delete
								</button>
							</li>
						)}
					</ul>
				</div>
			</td>
		</tr>
	);
});

QueueTableRow.displayName = "QueueTableRow";
