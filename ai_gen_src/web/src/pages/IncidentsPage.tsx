import { IncidentsTable } from '../components/IncidentsTable';

export function IncidentsPage() {
  return (
    <div className="page incidents-page">
      <section className="page__hero">
        <h1>Sự cố gần đây</h1>
        <p className="muted">Danh sách sự cố theo thời gian.</p>
      </section>

      <section className="page__section">
        <IncidentsTable paginated pageSize={15} />
      </section>
    </div>
  );
}
