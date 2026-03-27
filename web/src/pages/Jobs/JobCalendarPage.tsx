import React, { useEffect, useMemo, useState } from 'react';
import { Calendar, Badge, Modal, List, Card, DatePicker, Space, Tag, Button, message, Empty } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Api } from '../../api';
import type { Task, TaskExecution } from '../../api/modules/tasks';

const { RangePicker } = DatePicker;

interface ScheduleItem {
  id: string;
  jobId: string;
  status: string;
  scheduledTime: string;
  startTime?: string;
  endTime?: string;
  executionId: string;
}

const JobCalendarPage: React.FC = () => {
  const [schedules, setSchedules] = useState<ScheduleItem[]>([]);
  const [jobs, setJobs] = useState<Task[]>([]);
  const [selectedDate, setSelectedDate] = useState<Dayjs | null>(null);
  const [visible, setVisible] = useState(false);
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null]>([null, null]);
  const [loading, setLoading] = useState(true);

  const loadData = async () => {
    setLoading(true);
    try {
      const jobRes = await Api.tasks.getTaskList({ page: 1, pageSize: 200 });
      const jobsList = jobRes.data.list || [];
      setJobs(jobsList);

      const executionResponses = await Promise.all(
        jobsList.map((job) => Api.tasks.getTaskExecutions(job.id, { page: 1, pageSize: 30 }))
      );

      const allSchedules: ScheduleItem[] = [];
      executionResponses.forEach((response, idx) => {
        const job = jobsList[idx];
        (response.data.list || []).forEach((execution: TaskExecution) => {
          const time = execution.startTime || execution.createdAt;
          if (!time) return;
          allSchedules.push({
            id: `${job.id}-${execution.id}`,
            jobId: job.id,
            status: execution.status,
            scheduledTime: time,
            startTime: execution.startTime,
            endTime: execution.endTime,
            executionId: execution.id,
          });
        });
      });

      setSchedules(allSchedules);
    } catch (error) {
      message.error((error as Error).message || '加载日历数据失败');
      setSchedules([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const filteredSchedules = useMemo(() => {
    if (!dateRange[0] || !dateRange[1]) return schedules;
    const startDate = dateRange[0].startOf('day');
    const endDate = dateRange[1].endOf('day');
    return schedules.filter((schedule) => {
      const scheduleDate = dayjs(schedule.scheduledTime);
      return scheduleDate.valueOf() >= startDate.valueOf() && scheduleDate.valueOf() <= endDate.valueOf();
    });
  }, [schedules, dateRange]);

  const getListData = (value: Dayjs) => {
    const selectedDateStr = value.format('YYYY-MM-DD');
    return filteredSchedules
      .filter((s) => dayjs(s.scheduledTime).format('YYYY-MM-DD') === selectedDateStr)
      .map((s) => {
        const job = jobs.find((j) => j.id === s.jobId);
        const type = s.status === 'success' ? 'success' : s.status === 'failed' ? 'error' : s.status === 'running' ? 'processing' : 'default';
        return { type, content: job ? job.name : 'Unknown job', status: s.status, job, schedule: s };
      });
  };

  const dateCellRender = (value: Dayjs) => {
    const listData = getListData(value);
    return (
      <ul className="events">
        {listData.slice(0, 3).map((item, index) => (
          <li key={index}>
            <Badge
              status={item.type as any}
              text={<span className="cursor-pointer truncate" onClick={() => openDetailModal(item)}>{item.content}</span>}
            />
          </li>
        ))}
        {listData.length > 3 && <li><Tag>+{listData.length - 3}</Tag></li>}
      </ul>
    );
  };

  const openDetailModal = (item: ReturnType<typeof getListData>[number]) => {
    setSelectedDate(dayjs(item.schedule.scheduledTime));
    setVisible(true);
  };

  const getTaskStats = () => {
    const success = filteredSchedules.filter((s) => s.status === 'success').length;
    const running = filteredSchedules.filter((s) => s.status === 'running').length;
    const failed = filteredSchedules.filter((s) => s.status === 'failed').length;
    return { success, running, failed };
  };

  const stats = getTaskStats();

  return (
    <Card style={{ background: '#16213e', border: '1px solid #2d3748' }}>
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-6 space-y-4 md:space-y-0">
        <h2 className="text-xl font-bold text-white">任务日历视图</h2>

        <div className="flex flex-col sm:flex-row gap-3 w-full md:w-auto">
          <Space>
            <Badge status="success" text="成功" />
            <Badge status="processing" text="运行中" />
            <Badge status="error" text="失败" />
            <Button size="middle" onClick={loadData} loading={loading} icon={<ReloadOutlined />}>刷新</Button>
          </Space>
          <RangePicker onChange={(dates) => setDateRange(dates || [null, null])} />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <Card size="small"><div className="flex items-center"><Badge status="success" className="mr-2" /><span>成功: {stats.success}</span></div></Card>
        <Card size="small"><div className="flex items-center"><Badge status="processing" className="mr-2" /><span>运行中: {stats.running}</span></div></Card>
        <Card size="small"><div className="flex items-center"><Badge status="error" className="mr-2" /><span>失败: {stats.failed}</span></div></Card>
      </div>

      <Card className="calendar-card">
        <Calendar dateCellRender={dateCellRender} onSelect={setSelectedDate} value={selectedDate || dayjs()} />
      </Card>

      <Modal title={`日期: ${selectedDate ? selectedDate.format('YYYY年MM月DD日') : ''}`} open={visible} footer={null} onCancel={() => setVisible(false)} destroyOnClose width={860}>
        {selectedDate && (
          <>
            <div className="mb-4">
              <h3 className="font-medium text-lg text-white">当日任务执行记录</h3>
              <p className="text-sm text-gray-400">显示日期: {selectedDate.format('YYYY年MM月DD日')}</p>
            </div>

            <List
              dataSource={getListData(selectedDate)}
              locale={{ emptyText: <Empty description="暂无记录" /> }}
              renderItem={(item, index) => {
                const job = item.job;
                const schedule = item.schedule;
                let duration = 'N/A';
                if (schedule.startTime && schedule.endTime) {
                  const start = dayjs(schedule.startTime);
                  const end = dayjs(schedule.endTime);
                  duration = `${end.diff(start, 'seconds')}秒`;
                }
                return (
                  <List.Item
                    key={index}
                    actions={[<Tag color={item.status === 'success' ? 'green' : item.status === 'failed' ? 'red' : item.status === 'running' ? 'blue' : 'default'}>{item.status}</Tag>]}
                  >
                    <List.Item.Meta
                      title={<span>{job ? job.name : '未知任务'} ({job?.type || '-'}) <Tag color="blue" className="ml-2">#{schedule.executionId}</Tag></span>}
                      description={
                        <div>
                          <div><strong>触发时间:</strong> {dayjs(schedule.scheduledTime).format('HH:mm:ss')}</div>
                          <div><strong>开始:</strong> {schedule.startTime ? dayjs(schedule.startTime).format('HH:mm:ss') : '未开始'}</div>
                          <div><strong>结束:</strong> {schedule.endTime ? dayjs(schedule.endTime).format('HH:mm:ss') : '未结束'}</div>
                          <div><strong>执行时长:</strong> {duration}</div>
                        </div>
                      }
                    />
                  </List.Item>
                );
              }}
            />
          </>
        )}
      </Modal>
    </Card>
  );
};

export default JobCalendarPage;
