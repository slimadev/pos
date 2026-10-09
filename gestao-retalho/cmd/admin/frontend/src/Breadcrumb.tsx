import { ChevronRightIcon } from "@heroicons/react/24/outline";

export default function Breadcrumb({ itens }: { itens: { rotulo: string; onClick?: () => void }[] }) {
  return (
    <nav className="admin-breadcrumb" aria-label="breadcrumb">
      {itens.map((item, i) => (
        <span key={i} className="admin-breadcrumb-item">
          {item.onClick ? (
            <button className="admin-breadcrumb-link" onClick={item.onClick}>
              {item.rotulo}
            </button>
          ) : (
            <span className="admin-breadcrumb-atual">{item.rotulo}</span>
          )}
          {i < itens.length - 1 && <ChevronRightIcon className="admin-breadcrumb-chevron" aria-hidden="true" />}
        </span>
      ))}
    </nav>
  );
}
