import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IconPlus, IconRefresh } from '@tabler/icons-react';
import { useTasks, useTaskForm } from './hooks';
import { TaskCard, TaskEmptyState, TaskModal } from './components';

export const TasksPage: React.FC = () => {
  const { t } = useTranslation();
  const [showModal, setShowModal] = useState(false);

  const {
    tasks,
    remotes,
    agents,
    loading,
    fetchAll,
    deleteTask,
    toggleTaskActive,
    triggerTask,
    getAgentName,
    getAgentStatus,
    getRemoteName,
  } = useTasks();

  const taskForm = useTaskForm({
    remotes,
    onSuccess: () => {
      setShowModal(false);
      taskForm.resetForm();
    },
  });

  const handleCreate = () => {
    taskForm.openCreateModal();
    setShowModal(true);
  };

  const handleEdit = (task: Parameters<typeof taskForm.openEditModal>[0]) => {
    taskForm.openEditModal(task);
    setShowModal(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t('tasks.confirm_delete'))) return;
    const success = await deleteTask(id);
    if (!success) {
      alert(t('tasks.delete_failed'));
    }
  };

  const handleTrigger = async (taskId: string, agentId: string) => {
    const success = await triggerTask(taskId, agentId);
    if (success) {
      alert(t('tasks.triggered_success'));
    } else {
      alert(t('tasks.trigger_failed'));
    }
  };

  const handleCloseModal = () => {
    setShowModal(false);
    taskForm.resetForm();
  };

  if (loading) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <IconRefresh className="spinner text-primary mb-3" size={48} />
              <p className="text-muted">{t('common.loading')}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="row mb-3">
        <div className="col-12 d-flex justify-content-between align-items-center">
          <div className="d-flex gap-2">
            <button className="btn btn-outline-secondary" onClick={fetchAll}>
              <IconRefresh size={16} />
              <span className="ms-1">{t('common.refresh')}</span>
            </button>
          </div>
          <button className="btn btn-primary" onClick={handleCreate}>
            <IconPlus size={16} />
            <span className="ms-1">{t('tasks.create_task')}</span>
          </button>
        </div>
      </div>

      <div className="row row-deck row-cards">
        {tasks.length > 0 ? (
          tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              remoteName={getRemoteName(task.rclone_remote_id)}
              agents={agents}
              onEdit={handleEdit}
              onDelete={handleDelete}
              onToggleActive={toggleTaskActive}
              onTrigger={handleTrigger}
              getAgentName={getAgentName}
              getAgentStatus={getAgentStatus}
            />
          ))
        ) : (
          <TaskEmptyState onCreate={handleCreate} />
        )}
      </div>

      <TaskModal
        isOpen={showModal}
        isEditing={!!taskForm.editingTask}
        formData={taskForm.formData}
        submitting={taskForm.submitting}
        isS3Remote={taskForm.isS3Remote}
        isDatabaseSource={taskForm.isDatabaseSource}
        agents={agents}
        remotes={remotes}
        onClose={handleCloseModal}
        onSubmit={taskForm.handleSubmit}
        updateFormData={taskForm.updateFormData}
        toggleAgent={taskForm.toggleAgent}
        toggleRcloneArg={taskForm.toggleRcloneArg}
        browserOpen={taskForm.browserOpen}
        browserPath={taskForm.browserPath}
        browserParent={taskForm.browserParent}
        browserEntries={taskForm.browserEntries}
        browserLoading={taskForm.browserLoading}
        browserError={taskForm.browserError}
        onOpenBrowser={taskForm.openBrowser}
        onCloseBrowser={taskForm.closeBrowser}
        onNavigateBrowser={taskForm.navigateBrowser}
        onApplyBrowserPath={taskForm.applyBrowserPath}
        onSetBrowserPath={taskForm.setBrowserPath}
      />
    </>
  );
};

export default TasksPage;
