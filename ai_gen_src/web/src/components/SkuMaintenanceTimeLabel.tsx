import { maintenanceActiveTimes } from '../utils/maintenanceWindow';

type Props = {
  startsAt: string;
  endsAt: string;
  title?: string;
};

export default function SkuMaintenanceTimeLabel({ startsAt, endsAt, title }: Props) {
  const times = maintenanceActiveTimes(startsAt, endsAt);
  if (!times) {
    return (
      <div className="overview-table__sku-maint-label" title={title}>
        Bảo trì đang hoạt động
      </div>
    );
  }
  return (
    <div className="overview-table__sku-maint-label" title={title}>
      Bảo trì từ
      <br />
      {times.start}
      <br />
      -
      <br />
      {times.end}
    </div>
  );
}
