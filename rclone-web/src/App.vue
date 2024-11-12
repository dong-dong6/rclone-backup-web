<template>
  <div id="app">
    <router-view />
    <Footer/>
  </div>
</template>

<script>
import Footer from "@/Footer.vue";
export default {
  name: 'App',
  components:{
    Footer,
  }
};
</script>

<style>


#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  text-align: center;
  color: #2c3e50;
}
</style>
