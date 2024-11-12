<!-- src/components/Footer.vue -->
<template>
  <footer class="footer">
    <p>
      &copy; 2024 <a href="https://itdong.me" target="_blank">itdong</a>. All rights reserved.
    </p>
  </footer>
</template>

<script>
export default {
  name: 'Footer',
};
</script>

<style scoped>
.footer {
  position: fixed;
  bottom: 0;
  width: 100%;
  text-align: center;
  background-color: #f8f9fa;
  padding: 5px 0;
  border-top: 0.5px solid #e7e7e7;
}
.footer a {
  color: #007bff;
  text-decoration: none;
}
.footer a:hover {
  text-decoration: underline;
}
</style>