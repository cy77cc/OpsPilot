import React, { useState } from 'react';

interface ListPageTemplateProps {
  title: string;
  data: any[];
  columns: {
    key: string;
    title: string;
    render?: (row: any) => React.ReactNode;
  }[];
  loading?: boolean;
}

const ListPageTemplate: React.FC<ListPageTemplateProps> = ({
  title,
  data,
  columns,
  loading = false
}) => {
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const filteredData = statusFilter === 'all' 
    ? data 
    : data.filter(item => item.status === statusFilter);

  if (loading) {
    return (
      <div className="bg-bg-secondary rounded-lg p-6">
        <h2 className="text-lg font-semibold text-primary mb-4">{title}</h2>
        <div className="mb-4 animate-pulse">
          <div className="h-10 bg-gray-700 rounded mb-2"></div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-700">
                {columns.map(col => (
                  <th key={col.key} className="py-3 px-4 text-left">
                    <div className="h-4 bg-gray-700 rounded w-24"></div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {[1, 2, 3, 4, 5].map(i => (
                <tr key={i} className="border-b border-gray-800">
                  {columns.map(col => (
                    <td key={col.key} className="py-3 px-4">
                      <div className="h-4 bg-gray-700 rounded w-32"></div>
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-bg-secondary rounded-lg p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-semibold text-primary">{title}</h2>
        <div className="flex flex-wrap gap-2 mb-6">
          <button
            className={`px-3 py-1 rounded text-sm ${statusFilter === 'all' ? 'bg-brand-primary text-white' : 'bg-gray-700 text-secondary'}`}
            onClick={() => setStatusFilter('all')}
          >
            全部
          </button>
          <button
            className={`px-3 py-1 rounded text-sm ${statusFilter === 'RUNNING' ? 'bg-status-running text-white' : 'bg-gray-700 text-secondary'}`}
            onClick={() => setStatusFilter('RUNNING')}
          >
            运行中
          </button>
          <button
            className={`px-3 py-1 rounded text-sm ${statusFilter === 'WARNING' ? 'bg-status-warning text-white' : 'bg-gray-700 text-secondary'}`}
            onClick={() => setStatusFilter('WARNING')}
          >
            警告
          </button>
          <button
            className={`px-3 py-1 rounded text-sm ${statusFilter === 'ERROR' ? 'bg-status-error text-white' : 'bg-gray-700 text-secondary'}`}
            onClick={() => setStatusFilter('ERROR')}
          >
            故障
          </button>
          <button
            className={`px-3 py-1 rounded text-sm ${statusFilter === 'OFFLINE' ? 'bg-status-offline text-white' : 'bg-gray-700 text-secondary'}`}
            onClick={() => setStatusFilter('OFFLINE')}
          >
            离线
          </button>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-700">
              {columns.map(col => (
                <th key={col.key} className="py-3 px-4 text-left text-quaternary">
                  {col.title}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filteredData.map((item, index) => (
              <tr key={index} className="border-b border-gray-800 hover:bg-gray-800">
                {columns.map(col => (
                  <td key={col.key} className="py-3 px-4 text-secondary">
                    {col.render ? col.render(item) : item[col.key]}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default ListPageTemplate;
